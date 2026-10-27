package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"time"
)

type Maintenance struct{ db *gorm.DB }

func NewMaintenance(db *gorm.DB) *Maintenance { return &Maintenance{db: db} }
func (r *Maintenance) CredentialCiphertexts(ctx context.Context, after uint64, limit int) ([]models.CredentialCiphertext, error) {
	rows := []models.CredentialCiphertext{}
	err := r.db.WithContext(ctx).Table("api_credentials").Select("id,username,provider_token_encrypted").Where("id > ? AND provider_token_encrypted <> ''", after).Order("id").Limit(limit).Scan(&rows).Error
	return rows, err
}
func (r *Maintenance) MOCiphertexts(ctx context.Context, after uint64, limit int) ([]models.MOCiphertext, error) {
	rows := []models.MOCiphertext{}
	err := r.db.WithContext(ctx).Table("mo_events").Select("id,credential_id,event_key,encrypted_message").Where("id > ? AND encrypted_message <> '' AND content_purged_at IS NULL", after).Order("id").Limit(limit).Scan(&rows).Error
	return rows, err
}
func (r *Maintenance) ReplaceCredentialCiphertext(ctx context.Context, id uint64, old, next string) (bool, error) {
	result := r.db.WithContext(ctx).Table("api_credentials").Where("id = ? AND provider_token_encrypted = ?", id, old).Update("provider_token_encrypted", next)
	return result.RowsAffected == 1, result.Error
}
func (r *Maintenance) ReplaceMOCiphertext(ctx context.Context, id uint64, old, next string) (bool, error) {
	result := r.db.WithContext(ctx).Table("mo_events").Where("id = ? AND encrypted_message = ? AND content_purged_at IS NULL", id, old).Update("encrypted_message", next)
	return result.RowsAffected == 1, result.Error
}

// Retains fingerprints and event IDs. Active deliveries always prevent content removal.
func (r *Maintenance) PurgeMO(ctx context.Context, before time.Time, limit int, apply bool) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ids []uint64
		if err := tx.Raw(`SELECT e.id FROM mo_events e WHERE e.processed AND e.content_purged_at IS NULL AND e.received_at < ? AND NOT EXISTS (SELECT 1 FROM mo_callback_jobs j WHERE j.event_id=e.id AND j.state IN ('pending','processing')) ORDER BY e.received_at,e.id LIMIT ? FOR UPDATE OF e SKIP LOCKED`, before, limit).Scan(&ids).Error; err != nil {
			return err
		}
		count = int64(len(ids))
		if !apply || count == 0 {
			return nil
		}
		result := tx.Table("mo_events").Where("id IN ? AND content_purged_at IS NULL", ids).Updates(map[string]any{"encrypted_message": "", "content_purged_at": time.Now().UTC()})
		count = result.RowsAffected
		return result.Error
	})
	return count, err
}
