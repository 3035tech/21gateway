package services

import (
	"context"
	"strconv"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
)

type V1StatusRepository interface {
	Consume(context.Context, uint64, int) ([]models.SMSAttempt, error)
	ByDate(context.Context, uint64, time.Time, time.Time, string, string, int) ([]models.SMSAttempt, error)
}
type V1StatusService struct {
	repo     V1StatusRepository
	location *time.Location
}

func NewV1StatusService(repo V1StatusRepository) *V1StatusService {
	return &V1StatusService{repo: repo, location: legacyQueryLocation}
}

func (s *V1StatusService) formatBody(rows []models.SMSAttempt) dtos.LegacyStatusBody {
	body := dtos.LegacyStatusBody{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, Message: dtos.LegacyQueryOK, SentStatus: make([]dtos.LegacyStatusItem, 0, len(rows))}
	if len(rows) == 0 {
		body.Code = dtos.LegacyCodeNoRecords
		body.Message = dtos.LegacyQueryEmpty
		return body
	}
	for _, row := range rows {
		sentAt := row.CreatedAt
		if row.DispatchedAt != nil {
			sentAt = *row.DispatchedAt
		}
		body.SentStatus = append(body.SentStatus, dtos.LegacyStatusItem{CorrelationID: row.CorrelationID, Destination: row.Destination, SentDate: sentAt.In(s.location).Format(legacyQueryDateLayout), UnixSentDate: strconv.FormatInt(sentAt.Unix(), 10), Status: models.DeliveryStatus(row.DeliveryStatus)})
	}
	return body
}

func (s *V1StatusService) ByDate(ctx context.Context, credentialID uint64, timestamp int64, correlationID, destination string) (dtos.LegacyStatusBody, error) {
	start, end := legacyDayRange(timestamp)
	rows, err := s.repo.ByDate(ctx, credentialID, start, end, correlationID, destination, dtos.LegacyStatusPageSize+1)
	if err != nil {
		return dtos.LegacyStatusBody{}, dbError(err)
	}
	if len(rows) > dtos.LegacyStatusPageSize {
		return dtos.LegacyStatusBody{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return s.formatBody(rows), nil
}

func (s *V1StatusService) Consume(ctx context.Context, credentialID uint64) (dtos.LegacyStatusBody, error) {
	rows, err := s.repo.Consume(ctx, credentialID, dtos.LegacyStatusPageSize)
	if err != nil {
		return dtos.LegacyStatusBody{}, dbError(err)
	}
	return s.formatBody(rows), nil
}
