package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
)

func TestPostgresMessageTraceAndCompanyHealth(t *testing.T) {
	ctx, db := postgresFixture(t)
	company := models.Company{Name: "Trace company", Active: true}
	other := models.Company{Name: "Other company", Active: true}
	for _, value := range []*models.Company{&company, &other} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	credential := models.APICredential{CompanyID: &company.ID, Username: "trace-client", PasswordHash: "unused", Active: true, CallbacksEnabled: true, CallbackURL: "https://example.test/private-path"}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	providerID := "550e8400-e29b-41d4-a716-446655440000"
	attempt := models.SMSAttempt{CompanyID: &company.ID, CredentialID: credential.ID, Destination: "5511999999999", Encoding: models.SMSEncodingGSM7, State: models.SubmissionUnknown, FailureCode: apperrors.CodeUnknownOutcome, ProviderMessageID: &providerID}
	if err := db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	event := models.DeliveryEvent{EventKey: "zoug:trace", ProviderMessageID: providerID, Status: models.DeliveryUnknown, OccurredAt: time.Now().UTC(), AttemptID: &attempt.ID, Processed: true}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	job := models.CallbackJob{EventID: event.ID, AttemptID: attempt.ID, CredentialID: credential.ID, URL: credential.CallbackURL, Status: models.DeliverySent, State: models.CallbackFailed, Attempts: 3, LastError: models.CallbackHTTPError, AvailableAt: time.Now().UTC()}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	moEvent := models.MOEvent{EventKey: "zoug:reply", CredentialID: credential.ID, ProviderMessageID: providerID, AttemptID: &attempt.ID, EncryptedMessage: "encrypted-test-content", Fingerprint: "test-fingerprint", PhoneNumber: "5511999999999", ReceivedAt: time.Now().UTC(), Processed: true}
	if err := db.Create(&moEvent).Error; err != nil {
		t.Fatal(err)
	}
	moJob := models.MOCallbackJob{EventID: moEvent.ID, AttemptID: attempt.ID, CredentialID: credential.ID, URL: "https://example.test/private-mo", State: models.CallbackDelivered, Attempts: 1, AvailableAt: time.Now().UTC()}
	if err := db.Create(&moJob).Error; err != nil {
		t.Fatal(err)
	}
	messageService := NewMessageService(repositories.NewSMSAttempts(db))
	trace, err := messageService.Trace(ctx, attempt.ID)
	if err != nil || trace.SubmissionState != models.SubmissionUnknown || len(trace.Events) != 4 || trace.ProviderMessageID == nil || *trace.ProviderMessageID != providerID {
		t.Fatalf("trace=%+v err=%v", trace, err)
	}
	kinds := map[string]bool{}
	for _, stage := range trace.Events {
		kinds[stage.Kind] = true
	}
	if !kinds["zoug_delivery"] || !kinds["client_dlr"] || !kinds["mo"] || !kinds["client_mo"] {
		t.Fatalf("missing stages: %+v", trace.Events)
	}
	encoded, err := json.Marshal(trace)
	if err != nil || strings.Contains(string(encoded), "private-path") || strings.Contains(string(encoded), "private-mo") || strings.Contains(string(encoded), "encrypted-test-content") {
		t.Fatal("trace exposed callback destination or MO content", err)
	}
	if _, err := messageService.Trace(security.WithCompanyScope(ctx, []uint64{other.ID}), attempt.ID); err == nil {
		t.Fatal("another company can read message trace")
	}
	health, err := NewCompanyService(repositories.NewCompanies(db)).Health(ctx, company.PublicRef)
	if err != nil || health.ActiveCredentials != 1 || health.DLRConfigured != 1 || health.UnknownSends != 1 || health.FailedCallbacks != 1 || health.LastProviderAt == nil || health.LastClientMOAt == nil {
		t.Fatalf("health=%+v err=%v", health, err)
	}
	if _, err := NewCompanyService(repositories.NewCompanies(db)).Health(security.WithCompanyScope(ctx, []uint64{other.ID}), company.PublicRef); err == nil {
		t.Fatal("another company can read health")
	}
	operations, err := NewOperationsService(repositories.NewOperations(db)).Snapshot(ctx)
	if err != nil || operations.UnresolvedSends != 1 {
		t.Fatalf("operations=%+v err=%v", operations, err)
	}
}
