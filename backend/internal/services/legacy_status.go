package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"strconv"
	"time"
)

type LegacyStatusRepository interface {
	Consume(context.Context, uint64, int) ([]models.SMSAttempt, error)
}
type LegacyStatusService struct {
	repo     LegacyStatusRepository
	location *time.Location
}

func NewLegacyStatusService(repo LegacyStatusRepository) *LegacyStatusService {
	return &LegacyStatusService{repo: repo, location: legacyQueryLocation}
}
func (s *LegacyStatusService) Consume(ctx context.Context, credentialID uint64) (dtos.LegacyStatusBody, error) {
	rows, err := s.repo.Consume(ctx, credentialID, dtos.LegacyStatusPageSize)
	if err != nil {
		return dtos.LegacyStatusBody{}, dbError(err)
	}
	body := dtos.LegacyStatusBody{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, Message: dtos.LegacyQueryOK, SentStatus: make([]dtos.LegacyStatusItem, 0, len(rows))}
	if len(rows) == 0 {
		body.Code = dtos.LegacyCodeNoRecords
		body.Message = dtos.LegacyQueryEmpty
	}
	for _, row := range rows {
		sentAt := row.CreatedAt
		if row.DispatchedAt != nil {
			sentAt = *row.DispatchedAt
		}
		body.SentStatus = append(body.SentStatus, dtos.LegacyStatusItem{CorrelationID: row.CorrelationID, Destination: row.Destination, SentDate: sentAt.In(s.location).Format(legacyQueryDateLayout), UnixSentDate: strconv.FormatInt(sentAt.Unix(), 10), Status: models.DeliveryStatus(row.DeliveryStatus)})
	}
	return body, nil
}
