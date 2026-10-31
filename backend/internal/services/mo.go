package services

import (
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"github.com/21mobile/sms-gateway/internal/providers/webhook"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

const MaximumMOMessageBytes = 4096
const MaximumMOEventKeyCharacters = 200

type MORepository interface {
	Accept(context.Context, models.MOEvent) error
}
type MOService struct {
	repo  MORepository
	vault *security.MOContentVault
}

func NewMOService(repo MORepository, vault *security.MOContentVault) *MOService {
	return &MOService{repo: repo, vault: vault}
}

// AcceptProviderMO links the incoming reply to its original MT. A missing MT
// remains retriable; the provider callback must not be acknowledged as stored.
func (s *MOService) AcceptProviderMO(ctx context.Context, provider models.Provider, message webhook.MOMessage) error {
	if provider == "" || message.ProviderMessageID == "" {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	lookup, ok := s.repo.(interface {
		FindOriginalForProvider(context.Context, models.Provider, string) (models.SMSAttempt, error)
	})
	if !ok {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	attempt, err := lookup.FindOriginalForProvider(ctx, provider, message.ProviderMessageID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if attempt.CredentialID == 0 {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	input := dtos.MOInput{}
	input.Body.EventKey = message.EventKey
	input.Body.CredentialID = attempt.CredentialID
	input.Body.ProviderMessageID = message.ProviderMessageID
	input.Body.PhoneNumber = attempt.Destination
	input.Body.Message = message.Message
	input.Body.ReceivedAt = message.ReceivedAt
	return s.acceptWithProvider(ctx, provider, input)
}
func (s *MOService) Accept(ctx context.Context, input dtos.MOInput) error {
	id, err := uuid.Parse(input.Body.ProviderMessageID)
	if err != nil || id == uuid.Nil {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	input.Body.ProviderMessageID = id.String()
	return s.acceptWithProvider(ctx, models.ProviderZoug, input)
}
func (s *MOService) acceptWithProvider(ctx context.Context, provider models.Provider, input dtos.MOInput) error {
	b := input.Body
	validPhone := len(b.PhoneNumber) > 0 && len(b.PhoneNumber) <= 15
	for _, c := range b.PhoneNumber {
		if c < '0' || c > '9' {
			validPhone = false
		}
	}
	if provider == "" || !smsprovider.ValidMessageID(b.ProviderMessageID) || b.CredentialID == 0 || strings.TrimSpace(b.EventKey) == "" || !utf8.ValidString(b.EventKey) || utf8.RuneCountInString(b.EventKey) > MaximumMOEventKeyCharacters || !utf8.ValidString(b.Message) || len(b.Message) == 0 || len(b.Message) > MaximumMOMessageBytes || !validPhone || b.ReceivedAt.IsZero() || b.ReceivedAt.After(time.Now().Add(5*time.Minute)) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	b.ReceivedAt = b.ReceivedAt.UTC().Truncate(time.Microsecond)
	canonical, err := json.Marshal(b)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	fingerprint, err := s.vault.Fingerprint(b.CredentialID, b.EventKey, canonical)
	if err != nil {
		return err
	}
	encrypted, err := s.vault.Encrypt(b.CredentialID, b.EventKey, b.Message)
	if err != nil {
		return err
	}
	event := models.MOEvent{EventKey: b.EventKey, CredentialID: b.CredentialID, ProviderCode: provider, ProviderMessageID: b.ProviderMessageID, EncryptedMessage: encrypted, Fingerprint: fingerprint, PhoneNumber: b.PhoneNumber, ReceivedAt: b.ReceivedAt}
	return dbError(s.repo.Accept(ctx, event))
}
