package services

import (
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"gorm.io/gorm"
)

func dbError(err error) error {
	if err == nil {
		return nil
	}
	var typed *apperrors.Error
	if errors.As(err, &typed) {
		return err
	}
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return apperrors.Wrap(apperrors.CodeNotFound, nil)
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	default:
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
}
func mapPage[A, B any](input dtos.Page[A], convert func(A) B) dtos.Page[B] {
	result := dtos.Page[B]{Items: make([]B, 0, len(input.Items)), Total: input.Total, Page: input.Page, PageSize: input.PageSize}
	for _, value := range input.Items {
		result.Items = append(result.Items, convert(value))
	}
	return result
}
