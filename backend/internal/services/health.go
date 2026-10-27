package services

import (
	"context"

	"github.com/21mobile/sms-gateway/internal/config"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

type HealthService struct{ ping func(context.Context) error }

func NewHealthService(ping func(context.Context) error) *HealthService {
	return &HealthService{ping: ping}
}

func (s *HealthService) CheckReadiness(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, config.ReadinessTimeout)
	defer cancel()
	if err := s.ping(ctx); err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	return nil
}
