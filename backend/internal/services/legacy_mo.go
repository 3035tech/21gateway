package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"strconv"
	"time"
	"unicode/utf8"
)

const MaximumLegacyMORecords = 1000
const MaximumLegacyQueryTimestamp int64 = 253402300799

type LegacyMORepository interface {
	QueryDay(context.Context, uint64, time.Time, time.Time, int) ([]models.MOEvent, error)
}
type LegacyMOService struct {
	repo  LegacyMORepository
	vault *security.MOContentVault
}

func NewLegacyMOService(repo LegacyMORepository, vault *security.MOContentVault) *LegacyMOService {
	return &LegacyMOService{repo: repo, vault: vault}
}
func (s *LegacyMOService) Query(ctx context.Context, credentialID uint64, timestamp int64) (dtos.LegacyMOBody, error) {
	if credentialID == 0 || timestamp < 0 || timestamp > MaximumLegacyQueryTimestamp {
		return dtos.LegacyMOBody{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	start, end := legacyDayRange(timestamp)
	ctx, cancel := context.WithTimeout(ctx, legacyQueryTimeout)
	defer cancel()
	rows, err := s.repo.QueryDay(ctx, credentialID, start, end, MaximumLegacyMORecords+1)
	if err != nil {
		return dtos.LegacyMOBody{}, dbError(err)
	}
	// Never return a silently truncated legacy list without a documented continuation contract.
	if len(rows) > MaximumLegacyMORecords {
		return dtos.LegacyMOBody{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	body := dtos.LegacyMOBody{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, Message: dtos.LegacyQueryOK, ListMO: make([]dtos.LegacyMOItem, 0, len(rows))}
	if len(rows) == 0 {
		body.Code = dtos.LegacyCodeNoRecords
		body.Message = dtos.LegacyQueryEmpty
	}
	for _, event := range rows {
		if event.CredentialID != credentialID || event.AttemptID == nil {
			return dtos.LegacyMOBody{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
		}
		message, err := s.vault.Decrypt(credentialID, event.EventKey, event.EncryptedMessage)
		if err != nil || !utf8.ValidString(message) || len(message) == 0 || len(message) > MaximumMOMessageBytes {
			return dtos.LegacyMOBody{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
		}
		body.ListMO = append(body.ListMO, dtos.LegacyMOItem{FromAddress: event.PhoneNumber, ReceivedDate: event.ReceivedAt.In(legacyQueryLocation).Format(legacyQueryDateLayout), UnixReceivedDate: strconv.FormatInt(event.ReceivedAt.Unix(), 10), Message: message, IDMO: strconv.FormatUint(event.ID, 10), IDMT: strconv.FormatUint(*event.AttemptID, 10)})
	}
	return body, nil
}
