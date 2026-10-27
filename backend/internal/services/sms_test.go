package services

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type memoryAttempts struct {
	created                []models.SMSAttempt
	completed              []models.SMSAttempt
	createErr, completeErr error
}

func (r *memoryAttempts) Create(ctx context.Context, attempt *models.SMSAttempt) error {
	if r.createErr != nil {
		return r.createErr
	}
	attempt.ID = uint64(len(r.created) + 1)
	r.created = append(r.created, *attempt)
	return nil
}
func (r *memoryAttempts) Complete(ctx context.Context, attempt models.SMSAttempt) error {
	r.completed = append(r.completed, attempt)
	return r.completeErr
}

type stubProvider struct {
	enabled  bool
	inputs   []whitelabel.SendRequest
	response whitelabel.SendResponse
	err      error
}

func (p *stubProvider) Enabled() bool { return p.enabled }
func (p *stubProvider) Send(ctx context.Context, input whitelabel.SendRequest) (whitelabel.SendResponse, error) {
	p.inputs = append(p.inputs, input)
	return p.response, p.err
}

func validSMS() dtos.LegacySMS {
	return dtos.LegacySMS{MessageText: "Ola, seu codigo e 1234", Destination: "11987654321", CorrelationID: "customer-ref"}
}

func TestSendTranslatesAndPersistsAcceptance(t *testing.T) {
	repo := &memoryAttempts{}
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent}}
	result, err := NewSMSService(repo, provider).Send(context.Background(), 7, []dtos.LegacySMS{validSMS()})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Status != dtos.LegacyStatusOK || result[0].Code != dtos.LegacyCodeAccepted || result[0].IDSMS != "1" || result[0].CorrelationID != "customer-ref" {
		t.Fatalf("unexpected legacy response: %+v", result)
	}
	if len(provider.inputs) != 1 {
		t.Fatalf("provider called %d times", len(provider.inputs))
	}
	sent := provider.inputs[0]
	if sent.Destination != "5511987654321" || sent.Message != validSMS().MessageText || sent.ReferenceID != "customer-ref" || sent.Index != "1" || sent.Encoding != whitelabel.EncodingGSM7 || sent.CallbackLevel != whitelabel.CallbackNone {
		t.Fatalf("unexpected translation: %+v", sent)
	}
	if len(repo.completed) != 1 || repo.completed[0].State != models.SubmissionAccepted || repo.completed[0].CredentialID != 7 || *repo.completed[0].ProviderMessageID != provider.response.MessageID {
		t.Fatalf("unexpected persisted result: %+v", repo.completed)
	}
}

func TestImmediateSMSPersistsOnlyEncryptedHistory(t *testing.T) {
	repo := &memoryAttempts{}
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent}}
	vault, err := security.NewSMSHistoryVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("h", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	sender := NewSMSService(repo, provider)
	sender.SetHistoryVault(vault)
	if _, err := sender.Send(context.Background(), 7, []dtos.LegacySMS{validSMS()}); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || repo.created[0].ContentRef == nil || repo.created[0].EncryptedMessage == "" || strings.Contains(repo.created[0].EncryptedMessage, validSMS().MessageText) {
		t.Fatal("SMS text not safely retained")
	}
	plain, err := vault.Decrypt(7, *repo.created[0].ContentRef, repo.created[0].EncryptedMessage)
	if err != nil || plain != validSMS().MessageText {
		t.Fatal("stored content differs from submitted text", err)
	}
}

func TestInvalidInputNeverCallsProvider(t *testing.T) {
	scheduled := int64(1900000000)
	for _, tc := range []struct {
		name    string
		input   []dtos.LegacySMS
		message dtos.LegacyMessageCode
	}{
		{"empty batch", nil, dtos.LegacyInvalidRequest},
		{"oversized batch", make([]dtos.LegacySMS, whitelabel.MaximumBatchMessages+1), dtos.LegacyInvalidRequest},
		{"invalid phone", []dtos.LegacySMS{{Destination: "invalid", MessageText: "test"}}, dtos.LegacyInvalidNumber},
		{"empty text", []dtos.LegacySMS{{Destination: "11987654321", MessageText: "  "}}, dtos.LegacyInvalidMessage},
		{"long text", []dtos.LegacySMS{{Destination: "11987654321", MessageText: strings.Repeat("a", MaximumMessageCharacters+1)}}, dtos.LegacyInvalidMessage},
		{"scheduled", []dtos.LegacySMS{{Destination: "11987654321", MessageText: "test", ScheduledDate: &scheduled}}, dtos.LegacyScheduleUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &memoryAttempts{}
			provider := &stubProvider{enabled: true}
			results, err := NewSMSService(repo, provider).Send(context.Background(), 7, tc.input)
			if err != nil || results[0].Message != tc.message || results[0].Status != dtos.LegacyStatusNOK {
				t.Fatalf("result=%+v err=%v", results, err)
			}
			if len(provider.inputs) != 0 || len(repo.created) != 0 {
				t.Fatal("invalid input reached external/persistence side effects")
			}
		})
	}
}

func TestProviderFailureDoesNotRetry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   apperrors.Code
		state  models.SubmissionState
		legacy dtos.LegacyMessageCode
	}{
		{"unknown acceptance", apperrors.CodeUnknownOutcome, models.SubmissionUnknown, dtos.LegacyUnknownOutcome},
		{"balance", apperrors.CodeInsufficientBalance, models.SubmissionRejected, dtos.LegacyInsufficientBalance},
		{"provider auth", apperrors.CodeProviderUnauthorized, models.SubmissionRejected, dtos.LegacyProviderUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &memoryAttempts{}
			provider := &stubProvider{enabled: true, err: apperrors.Wrap(tc.code, errors.New("private token"))}
			results, err := NewSMSService(repo, provider).Send(context.Background(), 7, []dtos.LegacySMS{validSMS()})
			if err != nil || results[0].Message != tc.legacy || len(provider.inputs) != 1 || repo.completed[0].State != tc.state {
				t.Fatalf("results=%+v err=%v attempts=%+v", results, err, repo.completed)
			}
		})
	}
}

func TestPersistenceBeforeSendAndAcceptanceAfterUpdateFailure(t *testing.T) {
	t.Run("creation failure prevents sending", func(t *testing.T) {
		repo := &memoryAttempts{createErr: errors.New("database unavailable")}
		provider := &stubProvider{enabled: true}
		_, err := NewSMSService(repo, provider).Send(context.Background(), 7, []dtos.LegacySMS{validSMS()})
		if err == nil || len(provider.inputs) != 0 {
			t.Fatal("send happened without a durable attempt")
		}
	})
	t.Run("acknowledged message stays accepted", func(t *testing.T) {
		repo := &memoryAttempts{completeErr: errors.New("database unavailable")}
		provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent}}
		results, err := NewSMSService(repo, provider).Send(context.Background(), 7, []dtos.LegacySMS{validSMS()})
		if err != nil || results[0].Status != dtos.LegacyStatusOK || len(provider.inputs) != 1 {
			t.Fatalf("result=%+v err=%v", results, err)
		}
	})
}

func TestRateLimitIsPerCredential(t *testing.T) {
	repo := &memoryAttempts{}
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent}}
	service := NewSMSService(repo, provider)
	_, _ = service.Send(context.Background(), 7, []dtos.LegacySMS{validSMS()})
	limited, err := service.Send(context.Background(), 7, []dtos.LegacySMS{validSMS()})
	if err != nil || limited[0].Code != dtos.LegacyCodeRateLimited || len(provider.inputs) != 1 {
		t.Fatalf("limit failed: %+v %v", limited, err)
	}
	other, err := service.Send(context.Background(), 8, []dtos.LegacySMS{validSMS()})
	if err != nil || other[0].Status != dtos.LegacyStatusOK || len(provider.inputs) != 2 {
		t.Fatal("one credential blocked another")
	}
}
