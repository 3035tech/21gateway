package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
)

type Audits struct{ *store }

func NewAudits(db *gorm.DB) *Audits { return &Audits{store: &store{db: db}} }
func (r *Audits) Audits(ctx context.Context, q dtos.AuditListQuery) (dtos.Page[models.AuditEvent], error) {
	result := dtos.Page[models.AuditEvent]{Items: []models.AuditEvent{}, Page: q.Page, PageSize: q.PageSize}
	if err := validatePage(dtos.ListQuery{Page: q.Page, PageSize: q.PageSize, Direction: q.Direction}); err != nil {
		return result, err
	}
	db := r.db.WithContext(ctx).Table("audit_events AS a").Joins("LEFT JOIN system_users AS u ON u.id=a.actor_id")
	if q.Search != "" {
		term := searchTerm(q.Search)
		db = db.Where("a.action ILIKE ? OR a.resource ILIKE ? OR u.name ILIKE ?", term, term, term)
	}
	if q.ActorID != 0 {
		db = db.Where("a.actor_id=?", q.ActorID)
	}
	if q.Action != "" {
		db = db.Where("a.action=?", q.Action)
	}
	if q.Resource != "" {
		db = db.Where("a.resource=?", q.Resource)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	column := map[string]string{"id": "a.id", "actorName": "u.name", "action": "a.action", "resource": "a.resource", "resourceId": "a.resource_id", "createdAt": "a.created_at"}[q.Sort]
	if q.Sort == "" {
		column = "a.created_at"
	}
	if column == "" {
		return result, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	direction := string(q.Direction)
	err := db.Select("a.*, COALESCE(u.name,'') AS actor_name").Order(column + " " + direction).Order("a.id " + direction).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
}
func (r *Audits) CreateEvent(ctx context.Context, actor *uint64, action, resource string, id *uint64) error {
	return r.db.WithContext(ctx).Create(&models.AuditEvent{ActorID: actor, Action: action, Resource: resource, ResourceID: id}).Error
}
