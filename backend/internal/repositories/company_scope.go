package repositories

import (
	"context"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/security"
	"gorm.io/gorm"
)

func scopedCompanies(db *gorm.DB, ctx context.Context, column string) *gorm.DB {
	scope := security.Scope(ctx)
	if scope.Restricted {
		return db.Where(column+" IN ?", scope.IDs)
	}
	return db
}

func requireCompany(ctx context.Context, id uint64) error {
	if !security.AllowsCompany(ctx, id) {
		return apperrors.Wrap(apperrors.CodeForbidden, nil)
	}
	return nil
}
