package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type store struct{ db *gorm.DB }

func (s *store) Create(ctx context.Context, value any) error {
	return s.db.WithContext(ctx).Create(value).Error
}
func (s *store) Save(ctx context.Context, value any) error {
	return s.db.WithContext(ctx).Save(value).Error
}
func (s *store) Audit(ctx context.Context, actor *uint64, action, resource string, id *uint64) error {
	return NewAudits(s.db).CreateEvent(ctx, actor, action, resource, id)
}
func listQuery(db *gorm.DB, q dtos.ListQuery, columns map[string]string) (*gorm.DB, error) {
	column := columns[q.Sort]
	if q.Sort == "" {
		column = columns["createdAt"]
	}
	if column == "" {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if q.Page < 1 || q.PageSize < 1 || q.PageSize > 100 {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if q.Direction != dtos.SortAscending && q.Direction != dtos.SortDescending {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	query := db.Order(clause.OrderByColumn{Column: clause.Column{Name: column}, Desc: q.Direction == dtos.SortDescending})
	if column != "id" {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: q.Direction == dtos.SortDescending})
	}
	return query.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize), nil
}
func searchTerm(search string) string {
	return "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(search) + "%"
}
func validatePage(q dtos.ListQuery) error {
	if q.Page < 1 || q.PageSize < 1 || q.PageSize > 100 || (q.Direction != dtos.SortAscending && q.Direction != dtos.SortDescending) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return nil
}

func (s *store) DB() *gorm.DB { return s.db }
