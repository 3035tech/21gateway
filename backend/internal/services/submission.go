package services

import (
	"context"
	"strconv"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
)

type SubmissionRepository interface {
	Submission(context.Context, uint64, uint64) (models.SMSAttempt, error)
}

type SubmissionService struct{ repo SubmissionRepository }

func NewSubmissionService(repo SubmissionRepository) *SubmissionService {
	return &SubmissionService{repo: repo}
}

func (s *SubmissionService) Status(ctx context.Context, credentialID, sendID uint64) (dtos.SMSSubmissionStatus, error) {
	attempt, err := s.repo.Submission(ctx, credentialID, sendID)
	if err != nil {
		return dtos.SMSSubmissionStatus{}, dbError(err)
	}
	return dtos.SMSSubmissionStatus{
		IDSMS:          strconv.FormatUint(attempt.ID, 10),
		State:          attempt.State,
		FailureCode:    string(attempt.FailureCode),
		DeliveryStatus: attempt.DeliveryStatus,
		CreatedAt:      attempt.CreatedAt,
		DispatchedAt:   attempt.DispatchedAt,
	}, nil
}
