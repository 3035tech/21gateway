package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/google/uuid"
	"strings"
	"time"
)

type CallbackService struct{ repository *repositories.Callbacks }

func NewCallbackService(repo *repositories.Callbacks) *CallbackService {
	return &CallbackService{repository: repo}
}

func (s *CallbackService) Accept(ctx context.Context, input dtos.DeliveryInput) error {
	id, err := uuid.Parse(input.Body.ProviderMessageID)
	if err != nil || id == uuid.Nil {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	input.Body.ProviderMessageID = id.String()
	return s.AcceptProviderDelivery(ctx, models.ProviderZoug, input)
}
func (s *CallbackService) AcceptProviderDelivery(ctx context.Context, provider models.Provider, input dtos.DeliveryInput) error {
	if provider == "" || !smsprovider.ValidMessageID(input.Body.ProviderMessageID) || strings.TrimSpace(input.Body.EventKey) == "" || !input.Body.Status.Valid() || input.Body.OccurredAt.IsZero() || input.Body.OccurredAt.After(time.Now().Add(5*time.Minute)) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	event := models.DeliveryEvent{EventKey: input.Body.EventKey, ProviderCode: provider, ProviderMessageID: input.Body.ProviderMessageID, Status: input.Body.Status, OccurredAt: input.Body.OccurredAt.UTC().Truncate(time.Microsecond)}
	return dbError(s.repository.Accept(ctx, event))
}
func (s *CallbackService) Callbacks(ctx context.Context, q dtos.CallbackListInput) (dtos.Page[dtos.CallbackView], error) {
	value, err := s.repository.Callbacks(ctx, q)
	return value, dbError(err)
}
func (s *CallbackService) Received(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.ReceivedCallbackView], error) {
	value, err := s.repository.Received(ctx, q)
	return value, dbError(err)
}
func (s *CallbackService) Retry(ctx context.Context, actor, id uint64) error {
	return dbError(s.repository.RetryCallback(ctx, actor, id))
}
