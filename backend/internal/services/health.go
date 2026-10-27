package services

import (
	"context"
	"sync"
	"time"

	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

type HealthService struct {
	ping      func(context.Context) error
	smtp      func(context.Context) error
	sqs       func(context.Context) error
	mu        sync.Mutex
	checkedAt time.Time
	cached    dtos.DependenciesResponse
}

func NewHealthService(ping func(context.Context) error) *HealthService {
	return &HealthService{ping: ping}
}

func (s *HealthService) WithDependencies(smtp, sqs func(context.Context) error) *HealthService {
	s.smtp, s.sqs = smtp, sqs
	return s
}

func (s *HealthService) CheckDependencies(ctx context.Context) dtos.DependenciesResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.checkedAt.IsZero() && time.Since(s.checkedAt) < config.DependencyCheckCacheTTL {
		return s.cached
	}
	result := dtos.DependenciesResponse{Status: dtos.HealthStatusOK}
	var checks sync.WaitGroup
	for _, item := range []struct {
		status *dtos.DependencyStatus
		probe  func(context.Context) error
	}{
		{&result.Database, s.ping},
		{&result.SMTP, s.smtp},
		{&result.SQS, s.sqs},
	} {
		checks.Add(1)
		go func() {
			defer checks.Done()
			*item.status = checkDependency(ctx, item.probe)
		}()
	}
	checks.Wait()
	if result.Database == dtos.DependencyUnavailable || result.SMTP == dtos.DependencyUnavailable || result.SQS == dtos.DependencyUnavailable {
		result.Status = dtos.HealthStatusDegraded
	}
	if ctx.Err() == nil {
		s.cached = result
		s.checkedAt = time.Now()
	}
	return result
}

func checkDependency(ctx context.Context, probe func(context.Context) error) dtos.DependencyStatus {
	if probe == nil {
		return dtos.DependencyDisabled
	}
	check, cancel := context.WithTimeout(ctx, config.DependencyCheckTimeout)
	defer cancel()
	if err := probe(check); err != nil {
		return dtos.DependencyUnavailable
	}
	return dtos.DependencyOK
}

func (s *HealthService) CheckReadiness(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, config.ReadinessTimeout)
	defer cancel()
	if err := s.ping(ctx); err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	return nil
}
