package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Companies struct{ *store }

func NewCompanies(db *gorm.DB) *Companies { return &Companies{store: &store{db: db}} }
func (r *Companies) Transaction(ctx context.Context, mutate func(*Companies) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return mutate(NewCompanies(tx)) })
}
func (r *Companies) Company(ctx context.Context, id uint64) (models.Company, error) {
	var company models.Company
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND archived_at IS NULL", id).First(&company).Error
	return company, err
}
func (r *Companies) Companies(ctx context.Context, q dtos.ListQuery) (dtos.Page[models.Company], error) {
	result := dtos.Page[models.Company]{Items: []models.Company{}, Page: q.Page, PageSize: q.PageSize}
	db := r.db.WithContext(ctx).Model(&models.Company{}).Where("archived_at IS NULL")
	if q.Search != "" {
		db = db.Where("name ILIKE ? OR reference ILIKE ?", searchTerm(q.Search), searchTerm(q.Search))
	}
	if q.Active.Set {
		db = db.Where("active = ?", q.Active.Value)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	db, err := listQuery(db, q, map[string]string{"id": "id", "name": "name", "reference": "reference", "active": "active", "createdAt": "created_at"})
	if err != nil {
		return result, err
	}
	err = db.Find(&result.Items).Error
	return result, err
}
