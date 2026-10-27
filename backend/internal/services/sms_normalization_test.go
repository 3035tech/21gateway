package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"strings"
	"testing"
)

func TestSMSTextNormalization(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Olá, ação! 😊", "Ola, acao!"},
		{"ÁÀÂÃÄ ÉÈÊË ÍÌÎÏ ÓÒÔÕÖ ÚÙÛÜ Ç ñ", "AAAAA EEEE IIII OOOOO UUUU C n"},
		{"Ola\u0301, ac\u0327a\u0303o!", "Ola, acao!"},
		{"Teste 👨‍👩‍👧‍👦 🇧🇷 👍🏽 1️⃣!", "Teste    !"},
		{"Seu código 1234: R$ 10,00 / € 2,00 # *", "Seu codigo 1234: R$ 10,00 / € 2,00 # *"},
		{"Olá\nMundo", "Ola\nMundo"},
		{"漢字", "漢字"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, code := prepareSMSText(tc.input)
			if code != apperrors.CodeNone || got != tc.want {
				t.Fatalf("got %q code %s; want %q", got, code, tc.want)
			}
		})
	}
	for _, text := range []string{string([]byte{0xff}), strings.Repeat("a", MaximumMessageCharacters) + "😊"} {
		if _, code := prepareSMSText(text); code != apperrors.CodeInvalidMessage {
			t.Fatal("invalid source accepted")
		}
	}
}
func TestSingleSMSNormalizesBeforeProviderAndEncoding(t *testing.T) {
	original := validSMS()
	original.MessageText = "Olá, ação! 😊"
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent}}
	repo := &memoryAttempts{}
	result, err := NewSMSService(repo, provider).Send(context.Background(), 7, []dtos.LegacySMS{original})
	if err != nil || result[0].Status != dtos.LegacyStatusOK || len(provider.inputs) != 1 || provider.inputs[0].Message != "Ola, acao!" || provider.inputs[0].Encoding != models.SMSEncodingGSM7 {
		t.Fatal("incorrect normalized submission", err, provider.inputs)
	}
	if original.MessageText != "Olá, ação! 😊" {
		t.Fatal("caller input mutated")
	}
	empty := validSMS()
	empty.MessageText = "😊👨‍👩‍👧‍👦"
	p := &stubProvider{enabled: true}
	r := &memoryAttempts{}
	result, err = NewSMSService(r, p).Send(context.Background(), 7, []dtos.LegacySMS{empty})
	if err != nil || result[0].Message != dtos.LegacyInvalidMessage || len(p.inputs) != 0 || len(r.created) != 0 {
		t.Fatal("empty normalized text sent")
	}
}
func TestBatchUsesSameNormalizationWithoutChangingCaller(t *testing.T) {
	messages := []dtos.LegacySMS{validSMS(), validSMS()}
	messages[0].MessageText = "Olá 😊"
	messages[1].MessageText = "Ação 🇧🇷"
	provider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: []whitelabel.BatchResult{{MessageID: "550e8400-e29b-41d4-a716-446655440001"}, {MessageID: "550e8400-e29b-41d4-a716-446655440002"}}}
	_, err := NewSMSService(&batchMemory{}, provider).Send(context.Background(), 7, messages)
	if err != nil || len(provider.batchInput.Messages) != 2 || provider.batchInput.Messages[0].Message != "Ola" || provider.batchInput.Messages[1].Message != "Acao" || provider.batchInput.Encoding != models.SMSEncodingGSM7 {
		t.Fatal("batch normalization mismatch", err, provider.batchInput)
	}
	if messages[0].MessageText != "Olá 😊" || messages[1].MessageText != "Ação 🇧🇷" {
		t.Fatal("caller batch mutated")
	}
}
