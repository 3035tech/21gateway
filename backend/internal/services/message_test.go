package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"strings"
	"testing"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
)

func TestDiagnosticDestinationMustMatchConfiguredHash(t *testing.T) {
	allowed := sha256.Sum256([]byte("11999999999"))
	service := NewMessageService(nil).WithDiagnosticDestinationHash(hex.EncodeToString(allowed[:]))
	for _, destination := range []string{"21999999999", "1199999999", "invalid"} {
		_, err := service.DiagnoseCompany(context.Background(), 1, "unused", destination)
		var typed *apperrors.Error
		if !errors.As(err, &typed) || typed.Code != apperrors.CodeTestDestinationNotAllowed {
			t.Fatalf("destination=%q err=%v", destination, err)
		}
	}
	_, err := service.DiagnoseCompany(context.Background(), 1, "unused", "11999999999")
	var typed *apperrors.Error
	if !errors.As(err, &typed) || typed.Code != apperrors.CodeTestNotConfigured {
		t.Fatalf("allowed destination rejected: %v", err)
	}
}

type companyTestProvider struct{ stubProvider }

func (p *companyTestProvider) PrepareCompany(_ context.Context, company models.Company) (SMSProvider, error) {
	if company.ProviderTokenEncrypted == "" {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	return &p.stubProvider, nil
}

func TestCompanyTokenTestCallsProviderWithoutCredentialOrAttempt(t *testing.T) {
	provider := &companyTestProvider{stubProvider: stubProvider{enabled: true, response: smsprovider.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: string(whitelabel.StatusSent)}}}
	attempts := &memoryAttempts{}
	sender := NewSMSService(attempts, provider)
	company := models.Company{ID: 12, Active: true, ProviderTokenEncrypted: "encrypted-token"}
	providerID, err := sender.SendCompanyTest(context.Background(), company, "11987654321", "Teste do token", "test-ref")
	if err != nil || providerID == "" || len(attempts.created) != 0 || len(provider.inputs) != 1 {
		t.Fatalf("providerID=%q err=%v attempts=%d calls=%d", providerID, err, len(attempts.created), len(provider.inputs))
	}
	if got := provider.inputs[0]; got.Destination != "5511987654321" || got.ReferenceID != "test-ref" || got.Message != "Teste do token" || got.CallbackLevel != smsprovider.CallbackNone {
		t.Fatalf("unexpected provider request: %+v", got)
	}
	if _, err := sender.SendCompanyTest(context.Background(), company, "11987654321", "Teste do token", "test-ref-2"); !hasCode(err, apperrors.CodeRateLimited) || len(provider.inputs) != 1 {
		t.Fatalf("rate limit failed: %v, calls=%d", err, len(provider.inputs))
	}
}

func TestPostgresCompanyTokenTestUsesPublicRefAndAudits(t *testing.T) {
	ctx, db := postgresFixture(t)
	company := models.Company{Name: "Empresa XYZ", Active: true, ProviderTokenEncrypted: "encrypted-token"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	user := models.SystemUser{Name: "Admin", Email: "company-test@example.test", PasswordHash: "unused", Role: models.RoleAdmin, Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	provider := &companyTestProvider{stubProvider: stubProvider{enabled: true, response: smsprovider.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: string(whitelabel.StatusSent)}}}
	service := NewMessageServiceWithTestSend(repositories.NewSMSAttempts(db), nil, repositories.NewAudits(db), NewSMSService(repositories.NewSMSAttempts(db), provider), "11987654321", false).WithCompanyTests(repositories.NewCompanies(db))
	if _, err := service.SendCompanyTest(ctx, user.ID, company.PublicRef, "11999999999", ""); !hasCode(err, apperrors.CodeTestDestinationNotAllowed) || len(provider.inputs) != 0 {
		t.Fatalf("wrong destination reached provider: %v", err)
	}
	result, err := service.SendCompanyTest(ctx, user.ID, company.PublicRef, "11987654321", "")
	if err != nil || result.SubmissionState != models.SubmissionAccepted || result.IDSMS != provider.response.MessageID || len(provider.inputs) != 1 || provider.inputs[0].Message != standardTestMessage(company.Name) {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, len(provider.inputs))
	}
	var audit models.AuditEvent
	if err := db.Where("actor_id=? AND action=? AND resource=? AND resource_id=?", user.ID, testSendAuditAction, string(security.ResourceCompanies), company.ID).First(&audit).Error; err != nil {
		t.Fatal("test action not audited", err)
	}
	var count int64
	if err := db.Model(&models.SMSAttempt{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("unexpected SMS attempts: %d, %v", count, err)
	}
	var recorded models.CompanyTestSend
	if err := db.Where("correlation_id = ?", result.CorrelationID).First(&recorded).Error; err != nil || recorded.CompanyID != company.ID || recorded.ActorID != user.ID || recorded.State != models.SubmissionAccepted || recorded.ProviderMessageID == nil || *recorded.ProviderMessageID != result.IDSMS {
		t.Fatalf("accepted test not recorded: %+v, %v", recorded, err)
	}
	if summary, err := repositories.NewSummary(db).Summary(ctx, company.ID); err != nil || summary.Accepted != 1 || summary.Unknown != 0 {
		t.Fatalf("company counts after accepted test: %+v, %v", summary, err)
	}
	if history, err := service.CompanyTests(ctx, company.PublicRef); err != nil || len(history) != 1 || history[0].CorrelationID != result.CorrelationID {
		t.Fatalf("company test history: %+v, %v", history, err)
	}
	provider.err = apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
	provider.response.MessageID = "550e8400-e29b-41d4-a716-446655440001"
	service.sender = NewSMSService(repositories.NewSMSAttempts(db), provider)
	uncertain, err := service.SendCompanyTest(ctx, user.ID, company.PublicRef, "11987654321", "")
	if err != nil || uncertain.SubmissionState != models.SubmissionUnknown || uncertain.CorrelationID == "" || uncertain.FailureCode != string(apperrors.CodeUnknownOutcome) {
		t.Fatalf("unknown provider outcome was lost: result=%+v err=%v", uncertain, err)
	}
	recorded = models.CompanyTestSend{}
	if err := db.Where("correlation_id = ?", uncertain.CorrelationID).First(&recorded).Error; err != nil || recorded.State != models.SubmissionUnknown || recorded.FailureCode != apperrors.CodeUnknownOutcome || recorded.ProviderMessageID == nil || *recorded.ProviderMessageID != uncertain.IDSMS {
		t.Fatalf("uncertain test not recorded: %+v, %v", recorded, err)
	}
	if summary, err := repositories.NewSummary(db).Summary(ctx, company.ID); err != nil || summary.Accepted != 1 || summary.Unknown != 1 {
		t.Fatalf("company counts after uncertain test: %+v, %v", summary, err)
	}
	if history, err := service.CompanyTests(ctx, company.PublicRef); err != nil || len(history) != 2 || history[0].CorrelationID != uncertain.CorrelationID {
		t.Fatalf("uncertain test history: %+v, %v", history, err)
	}
}

func TestPostgresCredentialTestSendUsesExistingLimitsAndAudit(t *testing.T) {
	ctx, db := postgresFixture(t)
	company := models.Company{Name: "Client test", Active: true}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	user := models.SystemUser{Name: "Admin", Email: "test-send-admin@example.test", PasswordHash: "unused", Role: models.RoleAdmin, Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	credential := models.APICredential{CompanyID: &company.ID, Username: "client-test", PasswordHash: "private-hash", Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubProvider{enabled: true, response: smsprovider.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: string(whitelabel.StatusSent)}}
	service := NewMessageServiceWithTestSend(repositories.NewSMSAttempts(db), repositories.NewCredentials(db), repositories.NewAudits(db), NewSMSServiceWithLimiter(repositories.NewSMSAttempts(db), provider, repositories.NewSendLimits(db)), "11987654321", false)
	if _, err := service.SendTest(ctx, user.ID, credential.ID, "11999999999", ""); !hasCode(err, apperrors.CodeTestDestinationNotAllowed) || len(provider.inputs) != 0 {
		t.Fatal("unauthorized destination reached provider", err)
	}
	result, err := service.SendTest(ctx, user.ID, credential.ID, "11987654321", "")
	if err != nil || result.SubmissionState != models.SubmissionAccepted || result.IDSMS == "" || len(provider.inputs) != 1 || provider.inputs[0].Message != standardTestMessage(company.Name) || provider.inputs[0].ReferenceID != result.CorrelationID {
		t.Fatalf("test send result=%+v err=%v calls=%d", result, err, len(provider.inputs))
	}
	var attempt models.SMSAttempt
	if err := db.Where("credential_id=?", credential.ID).First(&attempt).Error; err != nil || attempt.CompanyID == nil || *attempt.CompanyID != company.ID || attempt.State != models.SubmissionAccepted || attempt.CorrelationID != result.CorrelationID {
		t.Fatal("attempt not persisted", err, attempt)
	}
	var audit models.AuditEvent
	if err := db.Where("actor_id=? AND action=? AND resource_id=?", user.ID, testSendAuditAction, credential.ID).First(&audit).Error; err != nil {
		t.Fatal("test action not audited", err)
	}
	if _, err := service.SendTest(ctx, user.ID, credential.ID, "11987654321", ""); err != nil || len(provider.inputs) != 1 {
		t.Fatal("shared rate limit bypassed", err, len(provider.inputs))
	}
}

func TestPostgresCredentialTestSendAcceptsFreeDestinationAndMessageWhenEnabled(t *testing.T) {
	ctx, db := postgresFixture(t)
	company := models.Company{Name: "Empresa XYZ", Active: true}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	user := models.SystemUser{Name: "Admin", Email: "free-test-send@example.test", PasswordHash: "unused", Role: models.RoleAdmin, Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	credential := models.APICredential{CompanyID: &company.ID, Username: "free-test-client", PasswordHash: "private-hash", Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubProvider{enabled: true, response: smsprovider.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: string(whitelabel.StatusSent)}}
	service := NewMessageServiceWithTestSend(repositories.NewSMSAttempts(db), repositories.NewCredentials(db), repositories.NewAudits(db), NewSMSServiceWithLimiter(repositories.NewSMSAttempts(db), provider, repositories.NewSendLimits(db)), "", true)
	for _, input := range []struct {
		destination, message string
		code                 apperrors.Code
	}{{"1199999999", "hello", apperrors.CodeInvalidDestination}, {"11999999999", " ", apperrors.CodeInvalidMessage}, {"11999999999", strings.Repeat("A", MaximumMessageCharacters+1), apperrors.CodeInvalidMessage}} {
		if _, err := service.SendTest(ctx, user.ID, credential.ID, input.destination, input.message); !hasCode(err, input.code) || len(provider.inputs) != 0 {
			t.Fatal("invalid test SMS reached provider", err)
		}
	}
	result, err := service.SendTest(ctx, user.ID, credential.ID, "11999999999", "Teste livre do token")
	if err != nil || result.SubmissionState != models.SubmissionAccepted || len(provider.inputs) != 1 || provider.inputs[0].Destination != "5511999999999" || provider.inputs[0].Message != "Teste livre do token" {
		t.Fatalf("custom test send result=%+v err=%v inputs=%+v", result, err, provider.inputs)
	}
}

func TestStandardTestMessageIncludesCompanyAndFitsSMS(t *testing.T) {
	if got := standardTestMessage("Empresa XYZ"); got != "Teste do token Zoug da empresa Empresa XYZ." {
		t.Fatal(got)
	}
	if got := standardTestMessage(strings.Repeat("A", MaximumMessageCharacters)); len(got) > MaximumMessageCharacters {
		t.Fatal("standard message exceeds SMS limit")
	}
}

func hasCode(err error, code apperrors.Code) bool {
	var typed *apperrors.Error
	return errors.As(err, &typed) && typed.Code == code
}
