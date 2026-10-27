package repositories

import (
	"context"
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserMFA struct{ db *gorm.DB }

func NewUserMFA(db *gorm.DB) *UserMFA { return &UserMFA{db: db} }
func (r *UserMFA) Transaction(ctx context.Context, fn func(*UserMFA) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(NewUserMFA(tx)) })
}
func (r *UserMFA) Get(ctx context.Context, id uint64, lock bool) (models.UserMFA, error) {
	var value models.UserMFA
	db := r.db.WithContext(ctx)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := db.Where("user_id = ?", id).First(&value).Error
	return value, err
}
func (r *UserMFA) Save(ctx context.Context, value models.UserMFA) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"secret_encrypted", "pending_encrypted", "pending_expires_at", "last_used_step", "updated_at"})}).Create(&value).Error
}
func (r *UserMFA) AddCodes(ctx context.Context, userID uint64, hashes [][]byte) error {
	rows := make([]models.UserMFARecoveryCode, 0, len(hashes))
	for _, hash := range hashes {
		rows = append(rows, models.UserMFARecoveryCode{UserID: userID, CodeHash: hash})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}
func (r *UserMFA) ClearCodes(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.UserMFARecoveryCode{}).Error
}
func (r *UserMFA) ConsumeCode(ctx context.Context, userID uint64, hash []byte) (bool, error) {
	result := r.db.WithContext(ctx).Where("user_id = ? AND code_hash = ?", userID, hash).Delete(&models.UserMFARecoveryCode{})
	return result.RowsAffected == 1, result.Error
}
func (r *UserMFA) AdvanceStep(ctx context.Context, userID uint64, step int64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&models.UserMFA{}).Where("user_id = ? AND last_used_step < ?", userID, step).Updates(map[string]any{"last_used_step": step, "updated_at": time.Now().UTC()})
	return result.RowsAffected == 1, result.Error
}
func (r *UserMFA) Audit(ctx context.Context, actor uint64, action string) error {
	return r.db.WithContext(ctx).Create(&models.AuditEvent{ActorID: &actor, Action: action, Resource: "users", ResourceID: &actor}).Error
}
func (r *UserMFA) RevokeSessions(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&models.SystemUser{}).Where("id = ? AND active AND archived_at IS NULL", id).UpdateColumn("session_version", gorm.Expr("session_version + 1")).Error
}
