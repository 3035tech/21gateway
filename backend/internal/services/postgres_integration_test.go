package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/21mobile/sms-gateway/internal/database"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// Every run owns an isolated schema. It never removes a caller's database or tables.
func TestPostgresBackofficeAndOutbox(t *testing.T) {
	ctx, isolated := postgresFixture(t)
	users, err := NewUserService(repositories.NewUsers(isolated))
	if err != nil {
		t.Fatal(err)
	}
	if err := users.BootstrapAdmin(ctx, "Admin", "admin@example.test", "strong-password-21"); err != nil {
		t.Fatal(err)
	}
	admin, err := users.Login(ctx, "admin@example.test", "strong-password-21")
	if err != nil {
		t.Fatal(err)
	}
	_, err = users.SaveUser(ctx, admin.ID, admin.ID, dtos.UserBody{Name: admin.Name, Email: admin.Email, Role: models.RoleViewer, Active: true})
	assertCode(t, err, apperrors.CodeLastAdmin)
	companies := NewCompanyService(repositories.NewCompanies(isolated))
	company, err := companies.SaveCompany(ctx, admin.ID, 0, dtos.CompanyBody{Name: "Example", Reference: "REF", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	vault, err := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	credentials := NewCredentialServiceWithVault(repositories.NewCredentials(isolated), vault)
	secret, err := credentials.SaveCredential(ctx, admin.ID, 0, dtos.CredentialBody{CompanyID: company.ID, Username: "test-api", Active: true, CallbacksEnabled: true, CallbackURL: "https://client.example/dlr"})
	if err != nil {
		t.Fatal(err)
	}
	if secret.Password == "" {
		t.Fatal("one-time password missing")
	}
	list := dtos.ListQuery{Page: 1, PageSize: 10, Direction: dtos.SortAscending}
	if values, err := credentials.Credentials(ctx, list); err != nil || values.Total != 1 {
		t.Fatal("credential list", values.Total, err)
	}
	rotated, err := credentials.RotateCredential(ctx, admin.ID, secret.Credential.ID)
	if err != nil || rotated.Password == secret.Password {
		t.Fatal("credential rotation", err)
	}
	token := "account-test-token"
	white := true
	allowed := []string{"203.0.113.10/32"}
	moURL := "https://client.example/mo"
	moEnabled := true
	if _, err := credentials.SaveCredential(ctx, admin.ID, secret.Credential.ID, dtos.CredentialBody{CompanyID: company.ID, Username: secret.Credential.Username, Active: true, CallbacksEnabled: true, CallbackURL: "https://client.example/dlr", ProviderToken: &token, WhitelistEnabled: &white, AllowedIPs: &allowed, MOCallbackURL: &moURL, MOCallbacksEnabled: &moEnabled}); err != nil {
		t.Fatal(err)
	}
	saved, err := repositories.NewCredentials(isolated).FindByUsername(ctx, secret.Credential.Username)
	if err != nil || len(saved.AllowedIPs) != 1 || saved.ProviderTokenEncrypted == token || saved.CallbackURL != "https://client.example/dlr" || !saved.CallbacksEnabled || saved.MOCallbackURL != moURL || !saved.MOCallbacksEnabled {
		t.Fatal("account settings persistence", err)
	}
	if plain, err := vault.Decrypt(saved.Username, saved.ProviderTokenEncrypted); err != nil || plain != token {
		t.Fatal("token decryption after persistence")
	}
	limiter := repositories.NewSendLimits(isolated)
	if ok, err := limiter.Allow(ctx, saved.ID, 200); err != nil || !ok {
		t.Fatal("shared limiter reservation", err)
	}
	if ok, err := limiter.Allow(ctx, saved.ID, 1); err != nil || ok {
		t.Fatal("shared request limit", err)
	}
	legacyPassword := "old"
	imported, err := credentials.SaveCredential(ctx, admin.ID, 0, dtos.CredentialBody{CompanyID: company.ID, Username: "old-client", Active: true, LegacyPassword: &legacyPassword})
	if err != nil || imported.Password != "" {
		t.Fatal("legacy password replacement", err)
	}
	legacySaved, err := repositories.NewCredentials(isolated).FindByUsername(ctx, "old-client")
	if err != nil {
		t.Fatal(err)
	}
	if match, err := security.VerifyPassword(legacyPassword, legacySaved.PasswordHash); err != nil || !match {
		t.Fatal("legacy authentication not preserved")
	}
	batchRepo := repositories.NewSMSAttempts(isolated)
	batchAttempts := []*models.SMSAttempt{{CredentialID: saved.ID, CompanyID: &company.ID, Destination: "invalid", Encoding: models.SMSEncodingGSM7, State: models.SubmissionPending}, {CredentialID: saved.ID, CompanyID: &company.ID, Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionPending}}
	if err := batchRepo.CreateBatch(ctx, batchAttempts); err != nil {
		t.Fatal(err)
	}
	completions := []models.SMSAttempt{*batchAttempts[0], *batchAttempts[1]}
	for i := range completions {
		completions[i].State = models.SubmissionRejected
		completions[i].FailureCode = apperrors.CodeProviderRejected
	}
	if err := batchRepo.CompleteBatch(ctx, completions); err != nil {
		t.Fatal("bulk outcome update", err)
	}
	// An early callback is persisted before a provider UUID is known locally.
	callbackRepo := repositories.NewCallbacks(isolated)
	callbacks := NewCallbackService(callbackRepo)
	providerID := uuid.NewString()
	input := dtos.DeliveryInput{}
	input.Body.EventKey = "delivery-1"
	input.Body.ProviderMessageID = providerID
	input.Body.Status = models.DeliveryDelivered
	input.Body.OccurredAt = time.Now().UTC().Truncate(time.Microsecond)
	if err := callbacks.Accept(ctx, input); err != nil {
		t.Fatal(err)
	}
	attempt := models.SMSAttempt{CredentialID: secret.Credential.ID, CompanyID: &company.ID, Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionAccepted, ProviderMessageID: &providerID}
	if err := repositories.NewSMSAttempts(isolated).Create(ctx, &attempt); err != nil {
		t.Fatal(err)
	}
	if err := callbackRepo.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if err := callbacks.Accept(ctx, input); err != nil {
		t.Fatal("duplicate event", err)
	}
	changed := input
	changed.Body.Status = models.DeliveryNotDelivered
	assertCode(t, callbacks.Accept(ctx, changed), apperrors.CodeConflict)
	jobs, err := callbacks.Callbacks(ctx, dtos.CallbackListInput{ListQuery: list})
	if err != nil || jobs.Total != 1 {
		t.Fatal("outbox must be unique", jobs.Total, err)
	}
	job, err := callbackRepo.Claim(ctx)
	if err != nil || job == nil {
		t.Fatal("queue claim", err)
	}
	if second, err := callbackRepo.Claim(ctx); err != nil || second != nil {
		t.Fatal("leased job reclaimed", err)
	}
	if err := callbackRepo.Finish(ctx, *job, true, "", 0); err != nil {
		t.Fatal(err)
	}
	if err := callbacks.Retry(ctx, admin.ID, job.ID); err == nil {
		t.Fatal("delivered callback retried")
	}
	// Archiving must preserve messages and their original company association.
	if err := credentials.Archive(ctx, admin.ID, secret.Credential.ID); err != nil {
		t.Fatal(err)
	}
	messages, err := NewMessageService(repositories.NewSMSAttempts(isolated)).Messages(ctx, list)
	if err != nil || messages.Total != 3 || messages.Items[0].CompanyID == nil || *messages.Items[0].CompanyID != company.ID {
		t.Fatal("history lost", err)
	}
	operator, err := users.SaveUser(ctx, admin.ID, 0, dtos.UserBody{Name: "Operator", Email: "operator@example.test", Role: models.RoleOperator, Active: true, Password: "another-strong-password"})
	if err != nil {
		t.Fatal(err)
	}
	logged, err := users.Login(ctx, operator.Email, "another-strong-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := users.SaveUser(ctx, admin.ID, operator.ID, dtos.UserBody{Name: operator.Name, Email: operator.Email, Role: operator.Role, Active: false}); err != nil {
		t.Fatal(err)
	}
	_, err = users.SessionUser(ctx, operator.ID, logged.SessionVersion)
	assertCode(t, err, apperrors.CodeUnauthorized)
	summary, err := NewSummaryService(repositories.NewSummary(isolated)).Summary(ctx)
	if err != nil || summary.Accepted != 1 {
		t.Fatal("summary", err)
	}
	// Independent polling consumption: 100-item bound, tenant isolation and new states.
	now := time.Now().UTC().Truncate(time.Microsecond)
	rows := make([]models.SMSAttempt, 101)
	for i := range rows {
		rows[i] = models.SMSAttempt{CredentialID: legacySaved.ID, CorrelationID: "poll-ref", Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionAccepted, DeliveryStatus: string(models.DeliverySent), DeliveryUpdatedAt: &now}
	}
	if err := isolated.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	polling := repositories.NewLegacyStatuses(isolated)
	first, err := polling.Consume(ctx, legacySaved.ID, dtos.LegacyStatusPageSize)
	if err != nil || len(first) != 100 {
		t.Fatal("first poll", len(first), err)
	}
	second, err := polling.Consume(ctx, legacySaved.ID, dtos.LegacyStatusPageSize)
	if err != nil || len(second) != 1 {
		t.Fatal("second poll", len(second), err)
	}
	empty, err := polling.Consume(ctx, legacySaved.ID, dtos.LegacyStatusPageSize)
	if err != nil || len(empty) != 0 {
		t.Fatal("repeated poll", len(empty), err)
	}
	later := now.Add(time.Second)
	if err := isolated.Model(&models.SMSAttempt{}).Where("id = ?", rows[0].ID).Updates(map[string]any{"delivery_status": models.DeliveryDelivered, "delivery_updated_at": later}).Error; err != nil {
		t.Fatal(err)
	}
	polledChange, err := polling.Consume(ctx, legacySaved.ID, dtos.LegacyStatusPageSize)
	if err != nil || len(polledChange) != 1 || polledChange[0].DeliveryStatus != string(models.DeliveryDelivered) {
		t.Fatal("new status lost", err)
	}
	other, err := polling.Consume(ctx, saved.ID, dtos.LegacyStatusPageSize)
	if err != nil || len(other) != 0 {
		t.Fatal("cross-account poll", err)
	}
	// Two parallel polls must consume this update exactly once in total.
	if err := isolated.Model(&models.SMSAttempt{}).Where("id = ?", rows[1].ID).Updates(map[string]any{"delivery_status": models.DeliveryDelivered, "delivery_updated_at": later}).Error; err != nil {
		t.Fatal(err)
	}
	type pollResult struct {
		rows []models.SMSAttempt
		err  error
	}
	results := make(chan pollResult, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			rows, err := polling.Consume(ctx, legacySaved.ID, dtos.LegacyStatusPageSize)
			results <- pollResult{rows: rows, err: err}
		}()
	}
	close(start)
	total := 0
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		total += len(result.rows)
	}
	if total != 1 {
		t.Fatalf("parallel polling returned %d states; want 1", total)
	}

	// MO early arrival, idempotency, outbox leases and current account authorization.
	if err := isolated.Model(&models.APICredential{}).Where("id = ?", legacySaved.ID).Updates(map[string]any{"mo_callbacks_enabled": true, "mo_callback_url": "https://client.example/mo"}).Error; err != nil {
		t.Fatal(err)
	}
	moRepo := repositories.NewMOs(isolated)
	moVault := testMOVault(t)
	moService := NewMOService(moRepo, moVault)
	moInput := dtos.MOInput{}
	moInput.Body.CredentialID = legacySaved.ID
	moInput.Body.EventKey = "early-mo-1"
	moInput.Body.ProviderMessageID = uuid.NewString()
	moInput.Body.PhoneNumber = "5511998877665"
	moInput.Body.Message = "private response\nOlá 😊"
	moInput.Body.ReceivedAt = time.Now().UTC().Truncate(time.Microsecond)
	if err := moService.Accept(ctx, moInput); err != nil {
		t.Fatal(err)
	}
	if err := moService.Accept(ctx, moInput); err != nil {
		t.Fatal("MO duplicate", err)
	}
	conflicting := moInput
	conflicting.Body.Message = "changed"
	assertCode(t, moService.Accept(ctx, conflicting), apperrors.CodeConflict)
	if claimed, err := moRepo.Claim(ctx); err != nil || claimed != nil {
		t.Fatal("unmatched MO delivered", err)
	}
	moUUID := moInput.Body.ProviderMessageID
	moAttempt := models.SMSAttempt{CredentialID: legacySaved.ID, CompanyID: &company.ID, ProviderMessageID: &moUUID, CorrelationID: "mo-ref", Destination: "5511998877665", Encoding: models.SMSEncodingGSM7, State: models.SubmissionAccepted}
	if err := isolated.Create(&moAttempt).Error; err != nil {
		t.Fatal(err)
	}
	if err := moRepo.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if err := moRepo.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	moJob, err := moRepo.Claim(ctx)
	if err != nil || moJob == nil {
		t.Fatal("MO outbox missing", err)
	}
	if duplicate, err := moRepo.Claim(ctx); err != nil || duplicate != nil {
		t.Fatal("MO leased twice", err)
	}
	event, original, moAllowed, err := moRepo.DeliveryData(ctx, *moJob)
	if err != nil || !moAllowed || original.ID != moAttempt.ID || strings.Contains(event.EncryptedMessage, moInput.Body.Message) {
		t.Fatal("MO association or privacy", err)
	}
	text, err := moVault.Decrypt(event.CredentialID, event.EventKey, event.EncryptedMessage)
	if err != nil || text != moInput.Body.Message {
		t.Fatal("MO content", err)
	}
	stale := *moJob
	invalidLease := uuid.NewString()
	stale.LeaseToken = &invalidLease
	assertCode(t, moRepo.Finish(ctx, stale, true, "", 0), apperrors.CodeConflict)
	if err := isolated.Model(&models.APICredential{}).Where("id = ?", legacySaved.ID).Update("mo_callbacks_enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, allowed, err := moRepo.DeliveryData(ctx, *moJob); err != nil || allowed {
		t.Fatal("disabled MO authorization", err)
	}
	if err := moRepo.Finish(ctx, *moJob, false, models.CallbackAccessBlocked, 0); err != nil {
		t.Fatal(err)
	}
	if next, err := moRepo.Claim(ctx); err != nil || next != nil {
		t.Fatal("blocked MO requeued", err)
	}
	cross := moInput
	cross.Body.EventKey = "cross-account"
	cross.Body.CredentialID = saved.ID
	assertCode(t, moService.Accept(ctx, cross), apperrors.CodeConflict)

	// Queries remain available with MO delivery disabled and never consume jobs/events.
	queryMO := NewLegacyMOService(moRepo, moVault)
	for range 2 {
		result, err := queryMO.Query(ctx, legacySaved.ID, moInput.Body.ReceivedAt.Unix())
		if err != nil || len(result.ListMO) != 1 || result.ListMO[0].Message != moInput.Body.Message {
			t.Fatal("MO query lost content", err)
		}
	}
	historical := time.Date(2020, 1, 2, 12, 0, 0, 0, time.UTC)
	dayStart, dayEnd := legacyDayRange(historical.Unix())
	for i, at := range []time.Time{dayStart.Add(-time.Microsecond), dayStart, dayEnd.Add(-time.Microsecond), dayEnd} {
		boundary := moInput
		boundary.Body.EventKey = fmt.Sprintf("boundary-%d", i)
		boundary.Body.ReceivedAt = at
		if err := moService.Accept(ctx, boundary); err != nil {
			t.Fatal("MO boundary persist", err)
		}
	}
	result, err := queryMO.Query(ctx, legacySaved.ID, historical.Unix())
	if err != nil || len(result.ListMO) != 2 {
		t.Fatal("wrong day boundaries", len(result.ListMO), err)
	}
	otherResult, err := queryMO.Query(ctx, saved.ID, historical.Unix())
	if err != nil || len(otherResult.ListMO) != 0 {
		t.Fatal("cross-tenant MO query", err)
	}

	// Real SQL preparation for versioned rotation and 90-day content retention.
	oldProviderKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	oldMOKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("m", 32)))
	newKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("n", 32)))
	rotatingTokens, err := security.NewTokenVaultWithKeys(oldProviderKey, "next", map[string]string{"next": newKey})
	if err != nil {
		t.Fatal(err)
	}
	rotatingMO, err := security.NewMOContentVaultWithKeys(oldMOKey, "next", map[string]string{"next": newKey}, "")
	if err != nil {
		t.Fatal(err)
	}
	maintenanceRepo := repositories.NewMaintenance(isolated)
	maintenance := NewMaintenanceService(maintenanceRepo, rotatingTokens, rotatingMO)
	preview, err := maintenance.Rotate(ctx, false)
	if err != nil || preview.Eligible == 0 || preview.Changed != 0 {
		t.Fatal("rotation preview", err, preview)
	}
	applied, err := maintenance.Rotate(ctx, true)
	if err != nil || applied.Changed != preview.Eligible {
		t.Fatal("rotation SQL", err, applied)
	}
	again, err := maintenance.Rotate(ctx, true)
	if err != nil || again.Eligible != 0 {
		t.Fatal("rotation resume", err, again)
	}
	if err := moService.Accept(ctx, moInput); err != nil {
		t.Fatal("rotation broke deduplication", err)
	}
	historicalRows, err := moRepo.QueryDay(ctx, legacySaved.ID, dayStart, dayEnd, 10)
	if err != nil || len(historicalRows) != 2 {
		t.Fatal("historical fixtures", err)
	}
	pendingMO := models.MOCallbackJob{EventID: historicalRows[0].ID, AttemptID: moAttempt.ID, CredentialID: legacySaved.ID, URL: "https://client.example/mo", State: models.CallbackPending, AvailableAt: time.Now().UTC()}
	if err := isolated.Create(&pendingMO).Error; err != nil {
		t.Fatal(err)
	}
	count, err := maintenance.PurgeMO(ctx, 90, false)
	if err != nil || count != 3 {
		t.Fatal("retention dry-run", count, err)
	}
	count, err = maintenance.PurgeMO(ctx, 90, true)
	if err != nil || count != 3 {
		t.Fatal("retention applied", count, err)
	}
	var purged models.MOEvent
	if err := isolated.Where("content_purged_at IS NOT NULL").First(&purged).Error; err != nil || purged.EncryptedMessage != "" || purged.Fingerprint == "" {
		t.Fatal("content/tombstone retention", err)
	}
	remaining, err := NewLegacyMOService(moRepo, rotatingMO).Query(ctx, legacySaved.ID, historical.Unix())
	if err != nil || len(remaining.ListMO) != 1 {
		t.Fatal("pending content lost", err)
	}
	stats, err := NewOperationsService(repositories.NewOperations(isolated)).Snapshot(ctx)
	if err != nil || stats.MO.Pending != 1 || stats.MO.FailedLastHour < 1 {
		t.Fatal("operations snapshot", err, stats)
	}
}
func safeSchema(value string) string {
	result := ""
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			result += string(r)
		}
	}
	return result
}
func assertCode(t *testing.T, err error, code apperrors.Code) {
	t.Helper()
	var app *apperrors.Error
	if !errors.As(err, &app) || app.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}

func postgresFixture(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("REQUIRE_DATABASE_TESTS") == "true" {
			t.Fatal("TEST_DATABASE_URL required for database verification")
		}
		t.Skip("TEST_DATABASE_URL absent; real PostgreSQL integration not exercised")
	}
	// Validate before creating any database object.
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	db, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal("database pool unavailable")
	}
	t.Cleanup(func() { _ = pool.Close() })
	schema := "test_21mobile_" + uuid.NewString()
	schema = safeSchema(schema)
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
	})
	q := parsed.Query()
	q.Set("search_path", schema)
	parsed.RawQuery = q.Encode()
	isolated, err := database.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	isolatedPool, err := isolated.DB()
	if err != nil {
		t.Fatal("isolated database pool unavailable")
	}
	t.Cleanup(func() { _ = isolatedPool.Close() })
	if err := database.Migrate(isolated); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(isolated); err != nil {
		t.Fatal("migration retry", err)
	}

	return ctx, isolated
}
