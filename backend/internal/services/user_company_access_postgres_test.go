package services

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
)

func TestPostgresUserCompanyAccess(t *testing.T) {
	ctx, db := postgresFixture(t)
	users, err := NewUserService(repositories.NewUsers(db))
	if err != nil {
		t.Fatal(err)
	}
	if err := users.BootstrapAdmin(ctx, "Admin", "admin-scope@example.test", "strong-password-21"); err != nil {
		t.Fatal(err)
	}
	admin, err := users.Login(ctx, "admin-scope@example.test", "strong-password-21")
	if err != nil {
		t.Fatal(err)
	}
	companies := NewCompanyService(repositories.NewCompanies(db))
	first, err := companies.SaveCompany(ctx, admin.ID, 0, dtos.CompanyBody{Name: "First", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := companies.SaveCompany(ctx, admin.ID, 0, dtos.CompanyBody{Name: "Second", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := users.SaveUser(ctx, admin.ID, 0, dtos.UserBody{Name: "No scope", Email: "no-scope@example.test", Role: models.RoleViewer, Active: true, Password: "another-strong-password"}); err == nil {
		t.Fatal("unscoped viewer was accepted")
	}
	viewer, err := users.SaveUser(ctx, admin.ID, 0, dtos.UserBody{Name: "Viewer", Email: "viewer-scope@example.test", Role: models.RoleViewer, Active: true, Password: "another-strong-password", CompanyIDs: []uint64{first.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(viewer.CompanyIDs) != 1 || viewer.CompanyIDs[0] != first.ID {
		t.Fatal("company link not returned")
	}
	account, err := repositories.NewUsers(db).User(ctx, viewer.ID)
	if err != nil {
		t.Fatal(err)
	}
	restricted := security.WithCompanyScope(ctx, account.CompanyIDs)
	list := dtos.ListQuery{Page: 1, PageSize: 10, Direction: dtos.SortAscending}
	visible, err := companies.Companies(restricted, list)
	if err != nil || visible.Total != 1 || visible.Items[0].ID != first.ID {
		t.Fatal("company list leaked", visible, err)
	}
	if _, err := companies.Company(restricted, second.ID); err == nil {
		t.Fatal("direct company lookup leaked")
	}
	if _, err := companies.SaveCompany(restricted, account.ID, 0, dtos.CompanyBody{Name: "Third", Active: true}); err == nil {
		t.Fatal("scoped user created a global company")
	}
	credentials := NewCredentialService(repositories.NewCredentials(db))
	firstCredential, err := credentials.SaveCredential(ctx, admin.ID, 0, dtos.CredentialBody{CompanyID: first.ID, Username: "first-scope", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	manualPassword := "manual-basic-auth-secret-21"
	secondCredential, err := credentials.SaveCredential(ctx, admin.ID, 0, dtos.CredentialBody{CompanyID: second.ID, Username: "second-scope", ManualPassword: &manualPassword, Active: true})
	if err != nil {
		t.Fatal(err)
	}
	basicAuth, err := NewAuthService(repositories.NewCredentials(db))
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct{ username, password string }{{firstCredential.Credential.Username, firstCredential.Password}, {secondCredential.Credential.Username, manualPassword}} {
		encoded := base64.StdEncoding.EncodeToString([]byte(pair.username + ":" + pair.password))
		if authenticated, err := basicAuth.Authenticate(ctx, "Basic "+encoded); err != nil || authenticated.Username != pair.username {
			t.Fatal("saved API credential failed Basic Auth", err)
		}
	}
	visibleCredentials, err := credentials.Credentials(restricted, list)
	if err != nil || visibleCredentials.Total != 1 || visibleCredentials.Items[0].ID != firstCredential.Credential.ID {
		t.Fatal("credential list leaked", visibleCredentials, err)
	}
	list.CompanyID = second.ID
	if filtered, err := credentials.Credentials(restricted, list); err != nil || filtered.Total != 0 {
		t.Fatal("company filter widened access", filtered, err)
	}
	if _, err := credentials.RotateCredential(restricted, viewer.ID, secondCredential.Credential.ID); err == nil {
		t.Fatal("credential rotation crossed company")
	}
	if _, err := credentials.Credential(restricted, secondCredential.Credential.ID); err == nil {
		t.Fatal("direct credential lookup crossed company")
	}
	if found, err := credentials.Credential(restricted, firstCredential.Credential.ID); err != nil || found.ID != firstCredential.Credential.ID {
		t.Fatal("direct credential lookup failed", err)
	}
	if _, err := credentials.SaveCredential(restricted, viewer.ID, 0, dtos.CredentialBody{CompanyID: second.ID, Username: "forbidden-scope", Active: true}); err == nil {
		t.Fatal("credential creation crossed company")
	}
	attempts := repositories.NewSMSAttempts(db)
	historyVault, err := security.NewSMSHistoryVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("h", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var firstAttemptID, secondAttemptID uint64
	for _, value := range []struct{ credentialID, companyID uint64 }{{firstCredential.Credential.ID, first.ID}, {secondCredential.Credential.ID, second.ID}} {
		attempt := models.SMSAttempt{CredentialID: value.credentialID, CompanyID: &value.companyID, Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionAccepted}
		reference := uuid.NewString()
		attempt.ContentRef = &reference
		attempt.EncryptedMessage, err = historyVault.Encrypt(value.credentialID, reference, "Mensagem de teste")
		if err != nil {
			t.Fatal(err)
		}
		if err := attempts.Create(ctx, &attempt); err != nil {
			t.Fatal(err)
		}
		if value.companyID == first.ID {
			firstAttemptID = attempt.ID
		} else {
			secondAttemptID = attempt.ID
		}
	}
	list.CompanyID = 0
	messages, err := NewMessageService(attempts).Messages(restricted, list)
	if err != nil || messages.Total != 1 || *messages.Items[0].CompanyID != first.ID {
		t.Fatal("message list leaked", messages, err)
	}
	messageService := NewMessageService(attempts).WithHistoryVault(historyVault)
	content, err := messageService.Content(restricted, firstAttemptID)
	if err != nil || content.State != dtos.MessageContentAvailable || content.MessageText != "Mensagem de teste" {
		t.Fatal("message content not available", content, err)
	}
	if _, err := messageService.Content(restricted, secondAttemptID); err == nil {
		t.Fatal("cross-company content exposed")
	}
	old := time.Now().UTC().Add(-91 * 24 * time.Hour)
	if err := db.Table("sms_attempts").Where("id = ?", firstAttemptID).Update("created_at", old).Error; err != nil {
		t.Fatal(err)
	}
	content, err = messageService.Content(restricted, firstAttemptID)
	if err != nil || content.State != dtos.MessageContentExpired || content.MessageText != "" {
		t.Fatal("expired text exposed", content, err)
	}
	if count, err := attempts.PurgeContent(ctx, time.Now().UTC().Add(-90*24*time.Hour), 10); err != nil || count != 1 {
		t.Fatal("content purge", count, err)
	}
	purged, err := attempts.MessageContent(restricted, firstAttemptID)
	if err != nil || purged.EncryptedMessage != "" || purged.ContentPurgedAt == nil {
		t.Fatal("ciphertext not purged", err)
	}
	summary, err := NewSummaryService(repositories.NewSummary(db)).Summary(restricted)
	if err != nil || summary.Companies != 1 || summary.ActiveCredentials != 1 || summary.Accepted != 1 {
		t.Fatal("summary leaked", summary, err)
	}
	updated, err := users.SaveUser(ctx, admin.ID, viewer.ID, dtos.UserBody{Name: viewer.Name, Email: viewer.Email, Role: models.RoleViewer, Active: true, CompanyIDs: []uint64{first.ID, second.ID}})
	if err != nil || len(updated.CompanyIDs) != 2 {
		t.Fatal("multiple company links", updated, err)
	}
	_, err = users.SessionUser(context.Background(), viewer.ID, account.SessionVersion)
	assertCode(t, err, apperrors.CodeUnauthorized)
}
