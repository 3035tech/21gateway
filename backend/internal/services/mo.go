package services

import (
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
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
