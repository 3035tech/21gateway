package repositories

import (
	"context"
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type V1Statuses struct{ db *gorm.DB }

func NewV1Statuses(db *gorm.DB) *V1Statuses { return &V1Statuses{db: db} }
func (r *V1Statuses) Consume(ctx context.Context, credentialID uint64, limit int) ([]models.SMSAttempt, error) {
	rows := []models.SMSAttempt{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serializes polling for this account without locking callback writers or sends.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", legacyPollingLockNamespace, int32(credentialID)).Error; err != nil {
			return err
		}
		if err := tx.Table("sms_attempts AS sms").Select("sms.*").Joins("LEFT JOIN legacy_status_receipts AS receipt ON receipt.attempt_id = sms.id").Where("sms.credential_id = ? AND sms.correlation_id <> '' AND sms.delivery_status <> '' AND sms.delivery_updated_at IS NOT NULL", credentialID).Where("receipt.attempt_id IS NULL OR receipt.delivery_updated_at <> sms.delivery_updated_at OR receipt.delivery_status <> sms.delivery_status").Order("sms.id ASC").Limit(limit).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		receipts := make([]models.SMSDeliveryReceipt, 0, len(rows))
		for _, row := range rows {
			receipts = append(receipts, models.SMSDeliveryReceipt{AttemptID: row.ID, CredentialID: credentialID, DeliveryUpdatedAt: *row.DeliveryUpdatedAt, DeliveryStatus: row.DeliveryStatus})
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "attempt_id"}}, DoUpdates: clause.AssignmentColumns([]string{"delivery_updated_at", "delivery_status"})}).Create(&receipts).Error
	})
	return rows, err
}

func (r *V1Statuses) ByDate(ctx context.Context, credentialID uint64, start, end time.Time, correlationID, destination string, limit int) ([]models.SMSAttempt, error) {
	rows := []models.SMSAttempt{}
	q := r.db.WithContext(ctx).Where("credential_id = ? AND created_at >= ? AND created_at < ?", credentialID, start, end)
	if correlationID != "" {
		q = q.Where("correlation_id = ?", correlationID)
	}
	if destination != "" {
		q = q.Where("destination = ?", destination)
	}
	err := q.Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

const legacyPollingLockNamespace = 210017
