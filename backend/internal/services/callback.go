package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/google/uuid"
	"strings"
	"time"
)

type CallbackService struct{ repository *repositories.Callbacks }

func NewCallbackService(repo *repositories.Callbacks) *CallbackService {
	return &CallbackService{repository: repo}
}

// AcceptWhiteLabelDelivery is the inactive adapter entry point. The external
// route must not be exposed until webhook authentication is confirmed.
func (s *CallbackService) AcceptWhiteLabelDelivery(ctx context.Context, payload []byte) error {
	input, err := whitelabel.NormalizeDeliveryCallback(payload)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, err)
	}
	return s.Accept(ctx, input)
}
func (s *CallbackService) Accept(ctx context.Context, input dtos.DeliveryInput) error {
	id, err := uuid.Parse(input.Body.ProviderMessageID)
	if err != nil || id == uuid.Nil || strings.TrimSpace(input.Body.EventKey) == "" || !input.Body.Status.Valid() || input.Body.OccurredAt.IsZero() || input.Body.OccurredAt.After(time.Now().Add(5*time.Minute)) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	event := models.DeliveryEvent{EventKey: input.Body.EventKey, ProviderMessageID: id.String(), Status: input.Body.Status, OccurredAt: input.Body.OccurredAt.UTC().Truncate(time.Microsecond)}
	return dbError(s.repository.Accept(ctx, event))
}
func (s *CallbackService) Callbacks(ctx context.Context, q dtos.CallbackListInput) (dtos.Page[dtos.CallbackView], error) {
	value, err := s.repository.Callbacks(ctx, q)
	return value, dbError(err)
}
func (s *CallbackService) Retry(ctx context.Context, actor, id uint64) error {
	return dbError(s.repository.RetryCallback(ctx, actor, id))
}
