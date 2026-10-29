package repositories

import (
	"context"
	"errors"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type MOs struct{ db *gorm.DB }

func NewMOs(db *gorm.DB) *MOs { return &MOs{db: db} }

// FindOriginal resolves the MO account through Zoug's MT message_id.
func (r *MOs) FindOriginal(ctx context.Context, providerMessageID string) (models.SMSAttempt, error) {
	var attempt models.SMSAttempt
	err := r.db.WithContext(ctx).Select("id,credential_id,destination,provider_message_id").Where("provider_message_id = ?", providerMessageID).First(&attempt).Error
	return attempt, err
}
func (r *MOs) Accept(ctx context.Context, event models.MOEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var credential models.APICredential
		if err := tx.First(&credential, event.CredentialID).Error; err != nil {
			return err
		}
		var other int64
		if err := tx.Model(&models.SMSAttempt{}).Where("provider_message_id = ? AND credential_id <> ?", event.ProviderMessageID, event.CredentialID).Count(&other).Error; err != nil {
			return err
		}
		if other != 0 {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		inserted := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "credential_id"}, {Name: "event_key"}}, DoNothing: true}).Create(&event)
		if inserted.Error != nil {
			return inserted.Error
		}
		if inserted.RowsAffected == 0 {
			var previous models.MOEvent
			if err := tx.Where("credential_id = ? AND event_key = ?", event.CredentialID, event.EventKey).First(&previous).Error; err != nil {
				return err
			}
			if previous.Fingerprint != event.Fingerprint {
				return apperrors.Wrap(apperrors.CodeConflict, nil)
			}
			return nil
		}
		return reconcileMO(tx, &event)
	})
}
func reconcileMO(tx *gorm.DB, event *models.MOEvent) error {
	var attempt models.SMSAttempt
	err := tx.Where("provider_message_id = ? AND credential_id = ?", event.ProviderMessageID, event.CredentialID).First(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	event.AttemptID = &attempt.ID
	event.Processed = true
	var credential models.APICredential
	if err := tx.First(&credential, event.CredentialID).Error; err != nil {
		return err
	}
	if credential.MOCallbacksEnabled && credential.MOCallbackURL != "" {
		job := models.MOCallbackJob{EventID: event.ID, AttemptID: attempt.ID, CredentialID: event.CredentialID, URL: credential.MOCallbackURL, State: models.CallbackPending, AvailableAt: time.Now().UTC()}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
	}
	return tx.Save(event).Error
}
func (r *MOs) Reconcile(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []models.MOEvent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED", Table: clause.Table{Name: "mo_events"}}).Where("NOT processed AND EXISTS (SELECT 1 FROM sms_attempts WHERE provider_message_id=mo_events.provider_message_id AND credential_id=mo_events.credential_id)").Order("received_at,id").Limit(CallbackBatchSize).Find(&events).Error; err != nil {
			return err
		}
		for i := range events {
			if err := reconcileMO(tx, &events[i]); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *MOs) Claim(ctx context.Context) (*models.MOCallbackJob, error) {
	var job models.MOCallbackJob
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("(state = ? AND available_at <= NOW()) OR (state = ? AND lease_until < NOW())", models.CallbackPending, models.CallbackProcessing).Order("id").First(&job).Error; err != nil {
			return err
		}
		token := uuid.NewString()
		lease := time.Now().UTC().Add(CallbackLease)
		job.LeaseToken = &token
		job.LeaseUntil = &lease
		job.State = models.CallbackProcessing
		job.Attempts++
		return tx.Save(&job).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}
func (r *MOs) DeliveryData(ctx context.Context, job models.MOCallbackJob) (models.MOEvent, models.SMSAttempt, bool, error) {
	var event models.MOEvent
	var attempt models.SMSAttempt
	if err := r.db.WithContext(ctx).Where("id = ? AND credential_id = ? AND attempt_id = ?", job.EventID, job.CredentialID, job.AttemptID).First(&event).Error; err != nil {
		return event, attempt, false, err
	}
	if err := r.db.WithContext(ctx).Where("id = ? AND credential_id = ?", job.AttemptID, job.CredentialID).First(&attempt).Error; err != nil {
		return event, attempt, false, err
	}
	var count int64
	err := r.db.WithContext(ctx).Table("api_credentials").Joins("LEFT JOIN companies ON companies.id=api_credentials.company_id").Where("api_credentials.id=? AND api_credentials.active AND api_credentials.archived_at IS NULL AND api_credentials.mo_callbacks_enabled AND api_credentials.mo_callback_url=? AND (api_credentials.company_id IS NULL OR (companies.active AND companies.archived_at IS NULL))", job.CredentialID, job.URL).Count(&count).Error
	return event, attempt, count == 1, err
}
func (r *MOs) Finish(ctx context.Context, job models.MOCallbackJob, delivered bool, code string, delay time.Duration) error {
	state := models.CallbackPending
	if delivered {
		state = models.CallbackDelivered
	} else if job.Attempts >= CallbackMaxAttempts || code == models.CallbackAccessBlocked || code == models.CallbackInvalidPayload {
		state = models.CallbackFailed
	}
	result := r.db.WithContext(ctx).Model(&models.MOCallbackJob{}).Where("id=? AND state=? AND lease_token=?", job.ID, models.CallbackProcessing, job.LeaseToken).Updates(map[string]any{"state": state, "last_error": code, "available_at": time.Now().UTC().Add(delay), "lease_until": nil, "lease_token": nil})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	return nil
}

// QueryDay is non-consuming and independent of callback delivery/enablement.
func (r *MOs) QueryDay(ctx context.Context, credentialID uint64, start, end time.Time, limit int) ([]models.MOEvent, error) {
	events := []models.MOEvent{}
	err := r.db.WithContext(ctx).Where("credential_id = ? AND attempt_id IS NOT NULL AND content_purged_at IS NULL AND received_at >= ? AND received_at < ?", credentialID, start, end).Order("received_at,id").Limit(limit).Find(&events).Error
	return events, err
}
