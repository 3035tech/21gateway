package services

import (
	"context"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type ScheduledListService struct{ repository *repositories.ScheduledSMS }

func NewScheduledListService(repository *repositories.ScheduledSMS) *ScheduledListService {
	return &ScheduledListService{repository: repository}
}

func (s *ScheduledListService) List(ctx context.Context, q dtos.ScheduledListQuery) (dtos.Page[dtos.ScheduledView], error) {
	var from, to *time.Time
	if q.DueFrom != "" {
		value, err := time.Parse(time.RFC3339, q.DueFrom)
		if err != nil {
			return dtos.Page[dtos.ScheduledView]{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		from = &value
	}
	if q.DueTo != "" {
		value, err := time.Parse(time.RFC3339, q.DueTo)
		if err != nil {
			return dtos.Page[dtos.ScheduledView]{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		to = &value
	}
	if from != nil && to != nil && from.After(*to) {
		return dtos.Page[dtos.ScheduledView]{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	values, err := s.repository.List(ctx, q, from, to)
	return values, dbError(err)
}
