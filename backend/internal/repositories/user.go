package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Users struct{ *store }

func NewUsers(db *gorm.DB) *Users { return &Users{store: &store{db: db}} }
func (r *Users) Transaction(ctx context.Context, mutate func(*Users) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return mutate(NewUsers(tx)) })
}

const userMutationLockID = 210015
const LoginLimit = 10

func (r *Users) LockUsers(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", userMutationLockID).Error
}
func (r *Users) UserByEmail(ctx context.Context, email string) (models.SystemUser, error) {
	var user models.SystemUser
	err := r.db.WithContext(ctx).Where("lower(email) = ? AND archived_at IS NULL", strings.ToLower(email)).First(&user).Error
	return user, err
}
func (r *Users) User(ctx context.Context, id uint64) (models.SystemUser, error) {
	var user models.SystemUser
	err := r.db.WithContext(ctx).Where("id = ? AND archived_at IS NULL", id).First(&user).Error
	return user, err
}
func (r *Users) AdminCount(ctx context.Context, except uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.SystemUser{}).Where("role = ? AND active AND archived_at IS NULL AND id <> ?", models.RoleAdmin, except).Count(&count).Error
	return count, err
}
func (r *Users) TouchLogin(ctx context.Context, user models.SystemUser, at time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.SystemUser{}).Where("id = ? AND session_version = ? AND active AND archived_at IS NULL", user.ID, user.SessionVersion).Update("last_login_at", at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	return nil
}
func (r *Users) UserCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.SystemUser{}).Count(&count).Error
	return count, err
}

func (r *Users) Users(ctx context.Context, q dtos.ListQuery) (dtos.Page[models.SystemUser], error) {
	result := dtos.Page[models.SystemUser]{Items: []models.SystemUser{}, Page: q.Page, PageSize: q.PageSize}
	db := r.db.WithContext(ctx).Model(&models.SystemUser{}).Select("id,name,email,role,active,created_at").Where("archived_at IS NULL")
	if q.Search != "" {
		db = db.Where("name ILIKE ? OR email ILIKE ?", searchTerm(q.Search), searchTerm(q.Search))
	}
	if q.Active.Set {
		db = db.Where("active = ?", q.Active.Value)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	db, err := listQuery(db, q, map[string]string{"id": "id", "name": "name", "email": "email", "role": "role", "active": "active", "createdAt": "created_at"})
	if err != nil {
		return result, err
	}
	err = db.Find(&result.Items).Error
	return result, err
}
func (r *Users) AllowLogin(ctx context.Context, key string) (bool, error) {
	return NewAuthenticationAttempts(r.db).Reserve(ctx, key, LoginLimit)
}
