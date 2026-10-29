package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Companies struct{ *store }

func NewCompanies(db *gorm.DB) *Companies { return &Companies{store: &store{db: db}} }
func (r *Companies) Transaction(ctx context.Context, mutate func(*Companies) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return mutate(NewCompanies(tx)) })
}
func (r *Companies) Company(ctx context.Context, id uint64) (models.Company, error) {
	var company models.Company
	err := scopedCompanies(r.db.WithContext(ctx), ctx, "id").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND archived_at IS NULL", id).First(&company).Error
	return company, err
}
func (r *Companies) CompanyView(ctx context.Context, id uint64) (models.Company, error) {
	var company models.Company
	err := scopedCompanies(r.db.WithContext(ctx), ctx, "id").Where("id = ? AND archived_at IS NULL", id).First(&company).Error
	return company, err
}
func (r *Companies) CompanyViewByPublicRef(ctx context.Context, publicRef string) (models.Company, error) {
	var company models.Company
	err := scopedCompanies(r.db.WithContext(ctx), ctx, "id").Where("public_ref = ? AND archived_at IS NULL", publicRef).First(&company).Error
	return company, err
}
func (r *Companies) Health(ctx context.Context, companyID uint64) (dtos.CompanyHealth, error) {
	var health dtos.CompanyHealth
	db := r.db.WithContext(ctx)
	var credentials struct {
		ActiveCredentials int64
		DLRConfigured     int64
		MOConfigured      int64
		ExpiringSoon      int64
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE active AND archived_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())) AS active_credentials,
		COUNT(*) FILTER (WHERE active AND archived_at IS NULL AND (expires_at IS NULL OR expires_at > NOW()) AND callbacks_enabled AND callback_url <> '') AS dlr_configured,
		COUNT(*) FILTER (WHERE active AND archived_at IS NULL AND (expires_at IS NULL OR expires_at > NOW()) AND mo_callbacks_enabled AND mo_callback_url <> '') AS mo_configured,
		COUNT(*) FILTER (WHERE active AND archived_at IS NULL AND expires_at > NOW() AND expires_at <= ?) AS expiring_soon
		FROM api_credentials WHERE company_id = ?`, time.Now().UTC().Add(7*24*time.Hour), companyID).Scan(&credentials).Error; err != nil {
		return health, err
	}
	health.ActiveCredentials = credentials.ActiveCredentials
	health.DLRConfigured = credentials.DLRConfigured
	health.MOConfigured = credentials.MOConfigured
	health.ExpiringSoon = credentials.ExpiringSoon
	if err := db.Table("sms_attempts").Where("company_id = ? AND state='unknown' AND COALESCE(delivery_status,'') IN ('','SENT','UNKNOWN')", companyID).Count(&health.UnknownSends).Error; err != nil {
		return health, err
	}
	var lastSend struct{ CreatedAt time.Time }
	if err := db.Table("sms_attempts").Select("created_at").Where("company_id = ?", companyID).Order("created_at DESC,id DESC").Limit(1).Scan(&lastSend).Error; err != nil {
		return health, err
	}
	if !lastSend.CreatedAt.IsZero() {
		health.LastSendAt = &lastSend.CreatedAt
	}
	var provider struct{ LastProviderAt *time.Time }
	if err := db.Raw(`SELECT MAX(received_at) AS last_provider_at FROM (
		SELECT e.created_at AS received_at FROM delivery_events e JOIN sms_attempts a ON a.id=e.attempt_id WHERE a.company_id=? AND e.event_key LIKE 'zoug:%'
		UNION ALL
		SELECT m.created_at AS received_at FROM mo_events m JOIN sms_attempts a ON a.id=m.attempt_id WHERE a.company_id=? AND m.event_key LIKE 'zoug:%'
	) callbacks`, companyID, companyID).Scan(&provider).Error; err != nil {
		return health, err
	}
	health.LastProviderAt = provider.LastProviderAt
	var dlr struct {
		Pending       int64
		Failed        int64
		LastDelivered *time.Time
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE j.state IN ('pending','processing')) AS pending, COUNT(*) FILTER (WHERE j.state='failed') AS failed, MAX(j.updated_at) FILTER (WHERE j.state='delivered') AS last_delivered FROM callback_jobs j JOIN sms_attempts a ON a.id=j.attempt_id WHERE a.company_id=?`, companyID).Scan(&dlr).Error; err != nil {
		return health, err
	}
	var mo struct {
		Pending       int64
		Failed        int64
		LastDelivered *time.Time
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE j.state IN ('pending','processing')) AS pending, COUNT(*) FILTER (WHERE j.state='failed') AS failed, MAX(j.updated_at) FILTER (WHERE j.state='delivered') AS last_delivered FROM mo_callback_jobs j JOIN sms_attempts a ON a.id=j.attempt_id WHERE a.company_id=?`, companyID).Scan(&mo).Error; err != nil {
		return health, err
	}
	health.PendingCallbacks = dlr.Pending + mo.Pending
	health.FailedCallbacks = dlr.Failed + mo.Failed
	health.LastClientDLRAt = dlr.LastDelivered
	health.LastClientMOAt = mo.LastDelivered
	return health, nil
}
func (r *Companies) Companies(ctx context.Context, q dtos.ListQuery) (dtos.Page[models.Company], error) {
	result := dtos.Page[models.Company]{Items: []models.Company{}, Page: q.Page, PageSize: q.PageSize}
	db := r.db.WithContext(ctx).Model(&models.Company{}).Where("archived_at IS NULL")
	db = scopedCompanies(db, ctx, "id")
	if q.CompanyID != 0 {
		db = db.Where("id = ?", q.CompanyID)
	}
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
