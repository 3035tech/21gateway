package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"time"
)

type OperationsRepository interface {
	Snapshot(context.Context) (dtos.OperationsSnapshot, error)
}
type OperationsService struct{ repo OperationsRepository }

func NewOperationsService(repo OperationsRepository) *OperationsService {
	return &OperationsService{repo: repo}
}
func (s *OperationsService) Snapshot(ctx context.Context) (dtos.OperationsSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := s.repo.Snapshot(ctx)
	return result, dbError(err)
}
