package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

func TestPostgresScheduledListIdentifiesClientAndTime(t *testing.T) {
	ctx, db := postgresFixture(t)
	company := models.Company{Name: "Client 21", Active: true}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	credential := models.APICredential{Username: "client-api", PasswordHash: "unused", CompanyID: &company.ID, Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewScheduledSMS(db)
	input := futureSMS()
	input.CorrelationID = "schedule-list-ref"
	if _, err := NewScheduledSMSService(repo, scheduledVault(t)).Enqueue(WithCredential(ctx, credential), credential.ID, []dtos.LegacySMS{input}); err != nil {
		t.Fatal(err)
	}
	service := NewScheduledListService(repo)
	query := dtos.ScheduledListQuery{Page: 1, PageSize: 10, Direction: dtos.SortAscending, CompanyID: company.ID, CredentialID: credential.ID, DueFrom: time.Unix(*input.ScheduledDate, 0).Add(-time.Minute).Format(time.RFC3339), DueTo: time.Unix(*input.ScheduledDate, 0).Add(time.Minute).Format(time.RFC3339)}
	page, err := service.List(ctx, query)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatal("scheduled listing", err, page.Total)
	}
	item := page.Items[0]
	if item.CompanyID == nil || *item.CompanyID != company.ID || item.CompanyName != company.Name || item.CredentialID != credential.ID || item.Username != credential.Username || item.CorrelationID != input.CorrelationID || item.ScheduledAt.Unix() != *input.ScheduledDate || item.State != models.ScheduleQueued || item.DispatchedAt != nil {
		t.Fatalf("wrong scheduling metadata: %+v", item)
	}
	messages, err := repositories.NewSMSAttempts(db).Messages(ctx, dtos.ListQuery{Page: 1, PageSize: 10, Direction: dtos.SortAscending, Sort: "companyName", CompanyID: company.ID})
	if err != nil || len(messages.Items) != 1 || messages.Items[0].CompanyName != company.Name {
		t.Fatal("message client name", err, messages)
	}
	event := models.DeliveryEvent{EventKey: "ui-audit-event", ProviderMessageID: "00000000-0000-4000-8000-000000000001", Status: models.DeliverySent, OccurredAt: time.Now().UTC(), AttemptID: &item.AttemptID}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	job := models.CallbackJob{EventID: event.ID, AttemptID: item.AttemptID, CredentialID: credential.ID, URL: "https://example.com/dlr", Status: models.DeliverySent, State: models.CallbackPending, AvailableAt: time.Now().UTC()}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	callbacks, err := repositories.NewCallbacks(db).Callbacks(ctx, dtos.CallbackListInput{ListQuery: dtos.ListQuery{Page: 1, PageSize: 10, Direction: dtos.SortAscending, Sort: "companyName", CompanyID: company.ID}})
	if err != nil || len(callbacks.Items) != 1 || callbacks.Items[0].CompanyName != company.Name {
		t.Fatal("callback client name", err, callbacks)
	}
	data, err := json.Marshal(item)
	if err != nil || strings.Contains(string(data), "Olá") || strings.Contains(string(data), "encryptedMessage") {
		t.Fatal("SMS content exposed in administration", err)
	}
	summary, err := NewSummaryService(repositories.NewSummary(db)).Summary(ctx)
	if err != nil || summary.ScheduledQueued != 1 || summary.ScheduledNext24h != 1 || summary.ScheduledDue != 0 || summary.ScheduledActive != 0 {
		t.Fatal("dashboard scheduling counts", err, summary)
	}
	query.State = models.ScheduleFailed
	page, err = service.List(ctx, query)
	if err != nil || page.Total != 0 {
		t.Fatal("state filter", err, page.Total)
	}
	query.DueFrom, query.DueTo = query.DueTo, query.DueFrom
	assertCode(t, func() error { _, err := service.List(ctx, query); return err }(), apperrors.CodeInvalidRequest)
}

func TestPostgresAdministrativeSortColumns(t *testing.T) {
	ctx, db := postgresFixture(t)
	base := dtos.ListQuery{Page: 1, PageSize: 10, Direction: dtos.SortAscending}
	for _, sort := range []string{"companyName", "deliveryStatus", "failureCode"} {
		query := base
		query.Sort = sort
		if _, err := repositories.NewSMSAttempts(db).Messages(ctx, query); err != nil {
			t.Fatalf("messages sort %s: %v", sort, err)
		}
	}
	for _, sort := range []string{"companyName", "url", "status", "lastError"} {
		query := dtos.CallbackListInput{ListQuery: base}
		query.Sort = sort
		if _, err := repositories.NewCallbacks(db).Callbacks(ctx, query); err != nil {
			t.Fatalf("callbacks sort %s: %v", sort, err)
		}
	}
	for _, sort := range []string{"actorName", "resourceId"} {
		query := dtos.AuditListQuery{Page: base.Page, PageSize: base.PageSize, Direction: base.Direction, Sort: sort}
		if _, err := repositories.NewAudits(db).Audits(ctx, query); err != nil {
			t.Fatalf("audit sort %s: %v", sort, err)
		}
	}
	for _, sort := range []string{"companyName", "providerTokenConfigured", "whitelistEnabled", "notifications"} {
		query := base
		query.Sort = sort
		if _, err := repositories.NewCredentials(db).Credentials(ctx, query); err != nil {
			t.Fatalf("credentials sort %s: %v", sort, err)
		}
	}
	for _, sort := range []string{"attemptId", "companyName", "username", "destination", "correlationId", "submissionState", "dispatchedAt"} {
		query := dtos.ScheduledListQuery{Page: 1, PageSize: 10, Direction: dtos.SortDescending, Sort: sort}
		if _, err := repositories.NewScheduledSMS(db).List(ctx, query, nil, nil); err != nil {
			t.Fatalf("schedules sort %s: %v", sort, err)
		}
	}
}

func TestPostgresAuditActorAndDomainFilters(t *testing.T) {
	ctx, db := postgresFixture(t)
	user := models.SystemUser{Name: "Marina Costa", Email: "marina@example.com", PasswordHash: "unused", Role: models.RoleAdmin, Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewAudits(db)
	resourceID := uint64(42)
	if err := repo.CreateEvent(ctx, &user.ID, "updated", "companies", &resourceID); err != nil {
		t.Fatal(err)
	}
	query := dtos.AuditListQuery{Page: 1, PageSize: 10, Direction: dtos.SortDescending, Sort: "actorName", Search: "Marina", ActorID: user.ID, Action: "updated", Resource: "companies"}
	page, err := repo.Audits(ctx, query)
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ActorName != user.Name {
		t.Fatal("audit actor/filter", err, page)
	}
	query.Resource = "credentials"
	page, err = repo.Audits(ctx, query)
	if err != nil || page.Total != 0 {
		t.Fatal("audit resource filter", err, page)
	}
}
