package services

import (
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

func TestPostgresProviderCallbackNamespaceAndOpaqueID(t *testing.T) {
	ctx, db := postgresFixture(t)
	credential := models.APICredential{Username: "namespace-test", PasswordHash: "unused", Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	const externalID = "opaque-42"
	other := models.Provider("test-supplier")
	zougAttempt := models.SMSAttempt{CredentialID: credential.ID, Destination: "5511987654321", Encoding: models.SMSEncodingGSM7, ProviderCode: models.ProviderZoug, ProviderMessageID: pointer(externalID), State: models.SubmissionAccepted}
	otherAttempt := models.SMSAttempt{CredentialID: credential.ID, Destination: "5511987654321", Encoding: models.SMSEncodingGSM7, ProviderCode: other, ProviderMessageID: pointer(externalID), State: models.SubmissionAccepted}
	if err := db.Create(&zougAttempt).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&otherAttempt).Error; err != nil {
		t.Fatal(err)
	}
	var delivery dtos.DeliveryInput
	delivery.Body.EventKey = "test-supplier:delivery-1"
	delivery.Body.ProviderMessageID = externalID
	delivery.Body.Status = models.DeliveryDelivered
	delivery.Body.OccurredAt = time.Now().UTC().Add(-time.Minute)
	if err := NewCallbackService(repositories.NewCallbacks(db)).AcceptProviderDelivery(ctx, other, delivery); err != nil {
		t.Fatal(err)
	}
	var event models.DeliveryEvent
	if err := db.Where("event_key=?", delivery.Body.EventKey).First(&event).Error; err != nil || event.AttemptID == nil || *event.AttemptID != otherAttempt.ID {
		t.Fatal("callback crossed provider namespace", err)
	}
	if err := db.First(&zougAttempt, zougAttempt.ID).Error; err != nil || zougAttempt.DeliveryStatus != "" {
		t.Fatal("other provider changed Zoug attempt", err)
	}
}

func pointer(value string) *string { return &value }
