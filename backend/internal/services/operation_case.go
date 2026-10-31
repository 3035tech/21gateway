package services

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type OperationCaseService struct{ repo *repositories.OperationCases }

func NewOperationCaseService(repo *repositories.OperationCases) *OperationCaseService {
	return &OperationCaseService{repo: repo}
}

func validOperationCaseKind(kind models.OperationCaseKind) bool {
	switch kind {
	case models.OperationUnknownSend, models.OperationOverdueSchedule, models.OperationDLRCallback, models.OperationMOCallback:
		return true
	}
	return false
}

func (s *OperationCaseService) List(ctx context.Context, q dtos.OperationCaseInput) (dtos.Page[dtos.OperationCaseItem], error) {
	if q.Kind != "" && !validOperationCaseKind(q.Kind) {
		return dtos.Page[dtos.OperationCaseItem]{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if q.Status != "" && !q.Status.Valid() {
		return dtos.Page[dtos.OperationCaseItem]{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if q.Page < 1 || q.PageSize < 1 || q.PageSize > models.OperationCaseMaxPageSize {
		return dtos.Page[dtos.OperationCaseItem]{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	items, err := s.repo.List(ctx, q)
	return items, dbError(err)
}

func (s *OperationCaseService) Update(ctx context.Context, actor uint64, kind models.OperationCaseKind, sourceID uint64, status models.OperationCaseStatus, note string) error {
	note = strings.TrimSpace(note)
	if !validOperationCaseKind(kind) || sourceID == 0 || !status.Valid() || utf8.RuneCountInString(note) > models.OperationCaseNoteMaxChars || (status != models.OperationCaseInProgress && note == "") {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return dbError(s.repo.Update(ctx, actor, kind, sourceID, status, note))
}

func (s *OperationCaseService) History(ctx context.Context, kind models.OperationCaseKind, sourceID uint64) ([]dtos.OperationCaseEvent, error) {
	if !validOperationCaseKind(kind) || sourceID == 0 {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	items, err := s.repo.History(ctx, kind, sourceID)
	return items, dbError(err)
}
