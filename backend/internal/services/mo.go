package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
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

// AcceptWhiteLabelMO links the incoming reply to the original MT before it is
// encrypted and persisted. A missing MT is retriable; never acknowledge it as
// received because Zoug discards callbacks after three unsuccessful attempts.
func (s *MOService) AcceptWhiteLabelMO(ctx context.Context, payload whitelabel.CallbackPayload) error {
	if payload.CallbackType != whitelabel.CallbackMO || payload.ShortMessage == nil {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	lookup, ok := s.repo.(interface {
		FindOriginal(context.Context, string) (models.SMSAttempt, error)
	})
	if !ok {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	attempt, err := lookup.FindOriginal(ctx, payload.MessageID)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	receivedAt, err := time.Parse(time.RFC3339Nano, payload.ReportDate)
	if err != nil || attempt.CredentialID == 0 {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, err)
	}
	identity, _ := json.Marshal(struct {
		MessageID string
		Date      string
		Message   string
	}{payload.MessageID, receivedAt.UTC().Format(time.RFC3339Nano), *payload.ShortMessage})
	digest := sha256.Sum256(identity)
	input := dtos.MOInput{}
	input.Body.EventKey = "zoug:" + hex.EncodeToString(digest[:])
	input.Body.CredentialID = attempt.CredentialID
	input.Body.ProviderMessageID = payload.MessageID
	input.Body.PhoneNumber = attempt.Destination
	input.Body.Message = *payload.ShortMessage
	input.Body.ReceivedAt = receivedAt
	return s.Accept(ctx, input)
}
func (s *MOService) Accept(ctx context.Context, input dtos.MOInput) error {
	b := input.Body
	id, err := uuid.Parse(b.ProviderMessageID)
	validPhone := len(b.PhoneNumber) > 0 && len(b.PhoneNumber) <= 15
	for _, c := range b.PhoneNumber {
		if c < '0' || c > '9' {
			validPhone = false
		}
	}
	if err != nil || id == uuid.Nil || b.CredentialID == 0 || strings.TrimSpace(b.EventKey) == "" || !utf8.ValidString(b.EventKey) || utf8.RuneCountInString(b.EventKey) > MaximumMOEventKeyCharacters || !utf8.ValidString(b.Message) || len(b.Message) == 0 || len(b.Message) > MaximumMOMessageBytes || !validPhone || b.ReceivedAt.IsZero() || b.ReceivedAt.After(time.Now().Add(5*time.Minute)) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	b.ProviderMessageID = id.String()
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
	event := models.MOEvent{EventKey: b.EventKey, CredentialID: b.CredentialID, ProviderMessageID: b.ProviderMessageID, EncryptedMessage: encrypted, Fingerprint: fingerprint, PhoneNumber: b.PhoneNumber, ReceivedAt: b.ReceivedAt}
	return dbError(s.repo.Accept(ctx, event))
}
