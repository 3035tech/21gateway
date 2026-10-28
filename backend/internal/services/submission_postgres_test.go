package services

import (
	"testing"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

func TestPostgresSubmissionStatusIsScopedToCredential(t *testing.T) {
	ctx, db := postgresFixture(t)
	credentials := []models.APICredential{{Username: "submission-a", PasswordHash: "unused", Active: true}, {Username: "submission-b", PasswordHash: "unused", Active: true}}
	if err := db.Create(&credentials).Error; err != nil {
		t.Fatal(err)
	}
	attempt := models.SMSAttempt{CredentialID: credentials[0].ID, Destination: "5511987654321", State: models.SubmissionPending}
	if err := db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSubmissionService(repositories.NewSMSAttempts(db))
	status, err := service.Status(ctx, credentials[0].ID, attempt.ID)
	if err != nil || status.State != models.SubmissionPending || status.IDSMS == "" {
		t.Fatal("own status unavailable", err)
	}
	if _, err := service.Status(ctx, credentials[1].ID, attempt.ID); !hasCode(err, apperrors.CodeNotFound) {
		t.Fatal("cross-credential status exposed", err)
	}
}
