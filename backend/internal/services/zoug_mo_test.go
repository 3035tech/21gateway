package services

import (
	"context"
	"errors"
	"testing"

	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
)

type zougMORepository struct {
	moMemory
	attempt   models.SMSAttempt
	lookupErr error
}

func (r *zougMORepository) FindOriginal(_ context.Context, _ string) (models.SMSAttempt, error) {
	return r.attempt, r.lookupErr
}

func TestZougMOLinksOriginalMTAndEncrypts(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	repo := &zougMORepository{attempt: models.SMSAttempt{CredentialID: 7, Destination: "5511998877665"}}
	service := NewMOService(repo, testMOVault(t))
	payload := whitelabel.CallbackPayload{SendType: "SMS", MessageID: id, CallbackType: whitelabel.CallbackMO, ShortMessage: new(string), ReportDate: "2026-10-28T13:14:08.000Z"}
	*payload.ShortMessage = "Obrigado"
	if err := service.AcceptWhiteLabelMO(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if len(repo.events) != 1 || repo.events[0].CredentialID != 7 || repo.events[0].PhoneNumber != "5511998877665" || repo.events[0].ProviderMessageID != id || repo.events[0].EncryptedMessage == "Obrigado" {
		t.Fatal("MO was not linked and encrypted")
	}
	repo.lookupErr = errors.New("missing MT")
	if err := service.AcceptWhiteLabelMO(context.Background(), payload); err == nil || len(repo.events) != 1 {
		t.Fatal("unmatched MO acknowledged")
	}
}
