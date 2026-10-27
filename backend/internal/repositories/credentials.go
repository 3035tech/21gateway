package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Credentials struct{ *store }

func NewCredentials(db *gorm.DB) *Credentials { return &Credentials{store: &store{db: db}} }
func (r *Credentials) FindByUsername(ctx context.Context, username string) (models.APICredential, error) {
	var result models.APICredential
	err := r.db.WithContext(ctx).Model(&models.APICredential{}).Select("api_credentials.*, companies.active AS company_active").Joins("LEFT JOIN companies ON companies.id = api_credentials.company_id").Where("api_credentials.username = ? AND api_credentials.archived_at IS NULL", username).First(&result).Error
	return result, err
}
func (r *Credentials) Create(ctx context.Context, credential *models.APICredential) error {
	if credential.CompanyID == nil {
		return requireCompany(ctx, 0)
	}
	if err := requireCompany(ctx, *credential.CompanyID); err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *Credentials) Transaction(ctx context.Context, mutate func(*Credentials) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return mutate(NewCredentials(tx)) })
}
func (r *Credentials) Credential(ctx context.Context, id uint64) (models.APICredential, error) {
	var credential models.APICredential
	err := r.credentialQuery(ctx).Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "api_credentials"}}).Where("api_credentials.id = ?", id).First(&credential).Error
	return credential, err
}
func (r *Credentials) ForTestSend(ctx context.Context, id uint64) (models.APICredential, error) {
	var credential models.APICredential
	err := r.credentialQuery(ctx).Where("api_credentials.id = ?", id).First(&credential).Error
	return credential, err
}
func (r *Credentials) CredentialView(ctx context.Context, id uint64) (models.APICredential, error) {
	var credential models.APICredential
	err := r.credentialQuery(ctx).Where("api_credentials.id = ?", id).First(&credential).Error
	return credential, err
}
func (r *Credentials) credentialQuery(ctx context.Context) *gorm.DB {
	return scopedCompanies(r.db.WithContext(ctx).Model(&models.APICredential{}).Select("api_credentials.*, companies.name AS company_name, companies.active AS company_active").Joins("LEFT JOIN companies ON companies.id = api_credentials.company_id").Where("api_credentials.archived_at IS NULL"), ctx, "api_credentials.company_id")
}
func (r *Credentials) Credentials(ctx context.Context, q dtos.ListQuery) (dtos.Page[models.APICredential], error) {
	result := dtos.Page[models.APICredential]{Items: []models.APICredential{}, Page: q.Page, PageSize: q.PageSize}
	db := r.credentialQuery(ctx).Select("api_credentials.id,api_credentials.company_id,companies.name AS company_name,api_credentials.username,api_credentials.active,api_credentials.callback_url,api_credentials.callbacks_enabled,api_credentials.created_at,api_credentials.expires_at,api_credentials.provider_token_encrypted,api_credentials.whitelist_enabled,api_credentials.allowed_ips,api_credentials.mo_callback_url,api_credentials.mo_callbacks_enabled,api_credentials.legacy_account_id")
	if q.Search != "" {
		db = db.Where("api_credentials.username ILIKE ?", searchTerm(q.Search))
	}
	if q.Active.Set {
		db = db.Where("api_credentials.active = ?", q.Active.Value)
	}
	if q.CompanyID != 0 {
		db = db.Where("api_credentials.company_id = ?", q.CompanyID)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	if err := validatePage(q); err != nil {
		return result, err
	}
	column := map[string]string{
		"id": "api_credentials.id", "username": "api_credentials.username", "companyName": "companies.name",
		"providerTokenConfigured": "(api_credentials.provider_token_encrypted <> '')",
		"whitelistEnabled":        "api_credentials.whitelist_enabled", "active": "api_credentials.active",
		"notifications": "(api_credentials.callbacks_enabled::int + api_credentials.mo_callbacks_enabled::int)",
		"createdAt":     "api_credentials.created_at",
		"expiresAt":     "api_credentials.expires_at",
	}[q.Sort]
	if q.Sort == "" {
		column = "api_credentials.created_at"
	}
	if column == "" {
		return result, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	direction := string(q.Direction)
	db = db.Order(column + " " + direction).Order("api_credentials.id " + direction).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize)
	err := db.Find(&result.Items).Error
	return result, err
}
