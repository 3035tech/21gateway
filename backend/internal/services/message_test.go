package services

import (
	"errors"
	"testing"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

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
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent}}
	service := NewMessageServiceWithTestSend(repositories.NewSMSAttempts(db), repositories.NewCredentials(db), repositories.NewAudits(db), NewSMSServiceWithLimiter(repositories.NewSMSAttempts(db), provider, repositories.NewSendLimits(db)), "11987654321")
	if _, err := service.SendTest(ctx, user.ID, credential.ID, "11999999999"); !hasCode(err, apperrors.CodeTestDestinationNotAllowed) || len(provider.inputs) != 0 {
		t.Fatal("unauthorized destination reached provider", err)
	}
	result, err := service.SendTest(ctx, user.ID, credential.ID, "11987654321")
	if err != nil || result.SubmissionState != models.SubmissionAccepted || result.IDSMS == "" || len(provider.inputs) != 1 || provider.inputs[0].Message != testSMSMessage || provider.inputs[0].ReferenceID != result.CorrelationID {
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
	if _, err := service.SendTest(ctx, user.ID, credential.ID, "11987654321"); err != nil || len(provider.inputs) != 1 {
		t.Fatal("shared rate limit bypassed", err, len(provider.inputs))
	}
}

func hasCode(err error, code apperrors.Code) bool {
	var typed *apperrors.Error
	return errors.As(err, &typed) && typed.Code == code
}
