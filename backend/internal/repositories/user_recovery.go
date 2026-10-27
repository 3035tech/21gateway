package repositories

import (
	"context"
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
)

type UserRecovery struct{ db *gorm.DB }

func NewUserRecovery(db *gorm.DB) *UserRecovery { return &UserRecovery{db: db} }
func (r *UserRecovery) Transaction(ctx context.Context, fn func(*UserRecovery) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(NewUserRecovery(tx)) })
}
func (r *UserRecovery) Store(ctx context.Context, id uint64, hash []byte, purpose string, expiry time.Time) error {
	return r.db.WithContext(ctx).Exec("INSERT INTO user_password_resets(user_id,token_hash,purpose,expires_at) VALUES (?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET token_hash=EXCLUDED.token_hash,purpose=EXCLUDED.purpose,expires_at=EXCLUDED.expires_at,created_at=NOW()", id, hash, purpose, expiry).Error
}
func (r *UserRecovery) ByHash(ctx context.Context, hash []byte) (models.UserPasswordReset, error) {
	var value models.UserPasswordReset
	err := r.db.WithContext(ctx).Where("token_hash = ? AND expires_at > NOW()", hash).First(&value).Error
	return value, err
}
func (r *UserRecovery) Consume(ctx context.Context, id uint64, hash []byte) (bool, error) {
	result := r.db.WithContext(ctx).Where("user_id = ? AND token_hash = ? AND expires_at > NOW()", id, hash).Delete(&models.UserPasswordReset{})
	return result.RowsAffected == 1, result.Error
}
func (r *UserRecovery) SetPassword(ctx context.Context, id uint64, hash string) error {
	result := r.db.WithContext(ctx).Model(&models.SystemUser{}).Where("id = ? AND active AND archived_at IS NULL", id).Updates(map[string]any{"password_hash": hash, "session_version": gorm.Expr("session_version + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *UserRecovery) Audit(ctx context.Context, id uint64, action string) error {
	return r.db.WithContext(ctx).Create(&models.AuditEvent{ActorID: &id, Action: action, Resource: "users", ResourceID: &id}).Error
}
