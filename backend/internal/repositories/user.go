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
	if err == nil {
		err = r.loadCompanyIDs(ctx, &user)
	}
	return user, err
}
func (r *Users) User(ctx context.Context, id uint64) (models.SystemUser, error) {
	var user models.SystemUser
	err := r.db.WithContext(ctx).Where("id = ? AND archived_at IS NULL", id).First(&user).Error
	if err == nil {
		err = r.loadCompanyIDs(ctx, &user)
	}
	return user, err
}
func (r *Users) UserByPublicRef(ctx context.Context, publicRef string) (models.SystemUser, error) {
	var user models.SystemUser
	err := r.db.WithContext(ctx).Where("public_ref = ? AND archived_at IS NULL", publicRef).First(&user).Error
	if err == nil {
		err = r.loadCompanyIDs(ctx, &user)
	}
	return user, err
}
func (r *Users) loadCompanyIDs(ctx context.Context, user *models.SystemUser) error {
	return r.db.WithContext(ctx).Table("system_user_companies").Where("user_id = ?", user.ID).Order("company_id").Pluck("company_id", &user.CompanyIDs).Error
}
func (r *Users) ReplaceCompanyIDs(ctx context.Context, userID uint64, companyIDs []uint64) error {
	if err := r.db.WithContext(ctx).Exec("DELETE FROM system_user_companies WHERE user_id = ?", userID).Error; err != nil {
		return err
	}
	for _, companyID := range companyIDs {
		if err := r.db.WithContext(ctx).Exec("INSERT INTO system_user_companies(user_id,company_id) VALUES (?,?)", userID, companyID).Error; err != nil {
			return err
		}
	}
	return nil
}
func (r *Users) ValidateCompanies(ctx context.Context, ids []uint64) error {
	var count int64
	if err := r.db.WithContext(ctx).Table("companies").Where("id IN ? AND active AND archived_at IS NULL", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return nil
}
func (r *Users) ReplaceOwnPassword(ctx context.Context, id uint64, version int64, oldHash, newHash string) error {
	result := r.db.WithContext(ctx).Model(&models.SystemUser{}).
		Where("id = ? AND session_version = ? AND password_hash = ? AND active AND archived_at IS NULL", id, version, oldHash).
		Updates(map[string]any{"password_hash": newHash, "session_version": version + 1})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	return nil
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
	db := r.db.WithContext(ctx).Model(&models.SystemUser{}).Select("id,public_ref,name,email,role,active,created_at").Where("archived_at IS NULL")
	if q.Search != "" {
		db = db.Where("name ILIKE ? OR email ILIKE ?", searchTerm(q.Search), searchTerm(q.Search))
	}
	if q.Active.Set {
		db = db.Where("active = ?", q.Active.Value)
	}
	if q.CompanyID != 0 {
		db = db.Where("EXISTS (SELECT 1 FROM system_user_companies access WHERE access.user_id = system_users.id AND access.company_id = ?)", q.CompanyID)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	db, err := listQuery(db, q, map[string]string{"id": "id", "name": "name", "email": "email", "role": "role", "active": "active", "createdAt": "created_at"})
	if err != nil {
		return result, err
	}
	err = db.Find(&result.Items).Error
	if err == nil {
		ids := make([]uint64, len(result.Items))
		index := make(map[uint64]int, len(ids))
		for i := range result.Items {
			ids[i], index[result.Items[i].ID] = result.Items[i].ID, i
		}
		if len(ids) > 0 {
			var links []struct {
				UserID    uint64
				CompanyID uint64
			}
			if err = r.db.WithContext(ctx).Table("system_user_companies").Select("user_id,company_id").Where("user_id IN ?", ids).Order("company_id").Scan(&links).Error; err != nil {
				return result, err
			}
			for _, link := range links {
				result.Items[index[link.UserID]].CompanyIDs = append(result.Items[index[link.UserID]].CompanyIDs, link.CompanyID)
			}
		}
	}
	return result, err
}
func (r *Users) AllowLogin(ctx context.Context, key string) (bool, error) {
	return NewAuthenticationAttempts(r.db).Reserve(ctx, key, LoginLimit)
}
