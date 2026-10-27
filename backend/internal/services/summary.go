package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type SummaryService struct{ repository *repositories.Summary }

func NewSummaryService(repo *repositories.Summary) *SummaryService {
	return &SummaryService{repository: repo}
}
func (s *SummaryService) Summary(ctx context.Context, companyID ...uint64) (dtos.Summary, error) {
	value, err := s.repository.Summary(ctx, companyID...)
	return value, dbError(err)
}
