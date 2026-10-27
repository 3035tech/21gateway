package repositories

import (
	"context"
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

const CallbackBatchSize = 32
const CallbackLease = 30 * time.Second
const CallbackMaxAttempts = 8

type Callbacks struct{ *store }

func NewCallbacks(db *gorm.DB) *Callbacks { return &Callbacks{store: &store{db: db}} }
func (r *Callbacks) Accept(ctx context.Context, event models.DeliveryEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		insert := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_key"}}, DoNothing: true}).Create(&event)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 0 {
			var existing models.DeliveryEvent
			if err := tx.Where("event_key = ?", event.EventKey).First(&existing).Error; err != nil {
				return err
			}
			if existing.ProviderMessageID != event.ProviderMessageID || existing.Status != event.Status || !existing.OccurredAt.Equal(event.OccurredAt) {
				return apperrors.Wrap(apperrors.CodeConflict, nil)
			}
			return nil
		}
		return reconcileEvent(tx, &event)
	})
}
func reconcileEvent(tx *gorm.DB, event *models.DeliveryEvent) error {
	var attempt models.SMSAttempt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("provider_message_id = ?", event.ProviderMessageID).First(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	event.AttemptID = &attempt.ID
	event.Processed = true
	current := models.DeliveryStatus(attempt.DeliveryStatus)
	newer := attempt.DeliveryUpdatedAt == nil || event.OccurredAt.After(*attempt.DeliveryUpdatedAt)
	if newer && !current.Terminal() {
		if err := tx.Model(&attempt).Updates(map[string]any{"delivery_status": event.Status, "delivery_updated_at": event.OccurredAt}).Error; err != nil {
			return err
		}
		var credential models.APICredential
		if err := tx.First(&credential, attempt.CredentialID).Error; err != nil {
			return err
		}
		if credential.CallbacksEnabled && credential.CallbackURL != "" {
			job := models.CallbackJob{EventID: event.ID, AttemptID: attempt.ID, CredentialID: credential.ID, URL: credential.CallbackURL, Status: event.Status, State: models.CallbackPending, AvailableAt: time.Now().UTC()}
			if err := tx.Create(&job).Error; err != nil {
				return err
			}
		}
	}
	return tx.Save(event).Error
}
func (r *Callbacks) Reconcile(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []models.DeliveryEvent
		// Only matched events enter this batch; early callbacks cannot starve later matches.
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED", Table: clause.Table{Name: "delivery_events"}}).Where("NOT processed AND EXISTS (SELECT 1 FROM sms_attempts WHERE provider_message_id = delivery_events.provider_message_id)").Order("occurred_at,id").Limit(CallbackBatchSize).Find(&events).Error
		if err != nil {
			return err
		}
		for i := range events {
			if err := reconcileEvent(tx, &events[i]); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r *Callbacks) Claim(ctx context.Context) (*models.CallbackJob, error) {
	var job models.CallbackJob
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where(`((state = ? AND available_at <= NOW()) OR (state = ? AND lease_until < NOW())) AND NOT EXISTS (SELECT 1 FROM callback_jobs older WHERE older.attempt_id=callback_jobs.attempt_id AND older.id<callback_jobs.id AND older.state IN (?,?))`, models.CallbackPending, models.CallbackProcessing, models.CallbackPending, models.CallbackProcessing).Order("id").First(&job).Error
		if err != nil {
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
func (r *Callbacks) DeliveryData(ctx context.Context, job models.CallbackJob) (models.SMSAttempt, bool, error) {
	var attempt models.SMSAttempt
	if err := r.db.WithContext(ctx).First(&attempt, job.AttemptID).Error; err != nil {
		return attempt, false, err
	}
	var count int64
	err := r.db.WithContext(ctx).Table("api_credentials").Joins("LEFT JOIN companies ON companies.id=api_credentials.company_id").Where("api_credentials.id=? AND api_credentials.active AND api_credentials.archived_at IS NULL AND api_credentials.callbacks_enabled AND api_credentials.callback_url=? AND (api_credentials.company_id IS NULL OR (companies.active AND companies.archived_at IS NULL))", job.CredentialID, job.URL).Count(&count).Error
	return attempt, count == 1, err
}
func (r *Callbacks) Finish(ctx context.Context, job models.CallbackJob, delivered bool, code string, delay time.Duration) error {
	state := models.CallbackPending
	if delivered {
		state = models.CallbackDelivered
	} else if job.Attempts >= CallbackMaxAttempts || code == models.CallbackAccessBlocked {
		state = models.CallbackFailed
	}
	result := r.db.WithContext(ctx).Model(&models.CallbackJob{}).Where("id=? AND state=? AND lease_token=?", job.ID, models.CallbackProcessing, job.LeaseToken).Updates(map[string]any{"state": state, "last_error": code, "available_at": time.Now().UTC().Add(delay), "lease_until": nil, "lease_token": nil})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	return nil
}
func (r *Callbacks) Callbacks(ctx context.Context, q dtos.CallbackListInput) (dtos.Page[dtos.CallbackView], error) {
	result := dtos.Page[dtos.CallbackView]{Items: []dtos.CallbackView{}, Page: q.Page, PageSize: q.PageSize}
	if err := validatePage(q.ListQuery); err != nil {
		return result, err
	}
	db := r.db.WithContext(ctx).Table("callback_jobs").Joins("JOIN sms_attempts ON sms_attempts.id=callback_jobs.attempt_id")
	db = scopedCompanies(db, ctx, "sms_attempts.company_id")
	if q.Search != "" {
		db = db.Where("callback_jobs.url ILIKE ?", searchTerm(q.Search))
	}
	if q.CompanyID != 0 {
		db = db.Where("sms_attempts.company_id=?", q.CompanyID)
	}
	if q.CallbackState != "" {
		db = db.Where("callback_jobs.state=?", q.CallbackState)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	column := map[string]string{"id": "callback_jobs.id", "attemptId": "callback_jobs.attempt_id", "companyName": "companies.name", "url": "callback_jobs.url", "status": "callback_jobs.status", "state": "callback_jobs.state", "attempts": "callback_jobs.attempts", "lastError": "callback_jobs.last_error", "createdAt": "callback_jobs.created_at"}[q.Sort]
	if q.Sort == "" {
		column = "callback_jobs.created_at"
	}
	if column == "" {
		return result, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	direction := string(q.Direction)
	err := db.Joins("LEFT JOIN companies ON companies.id=sms_attempts.company_id").Select("callback_jobs.*, sms_attempts.company_id, COALESCE(companies.name,'') AS company_name").Order(column + " " + direction).Order("callback_jobs.id " + direction).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
}
func (r *Callbacks) RetryCallback(ctx context.Context, actor, id uint64) error {
	return r.Transaction(ctx, func(repo *Callbacks) error {
		var job models.CallbackJob
		if err := repo.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&job, id).Error; err != nil {
			return err
		}
		var attempt models.SMSAttempt
		if err := repo.db.First(&attempt, job.AttemptID).Error; err != nil {
			return err
		}
		if attempt.CompanyID == nil {
			return requireCompany(ctx, 0)
		}
		if err := requireCompany(ctx, *attempt.CompanyID); err != nil {
			return err
		}
		if job.State != models.CallbackFailed || attempt.DeliveryStatus != string(job.Status) {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		var credential models.APICredential
		if err := repo.db.First(&credential, job.CredentialID).Error; err != nil {
			return err
		}
		if !credential.CallbacksEnabled || !credential.Active || credential.ArchivedAt != nil {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		if err := repo.db.Model(&job).Updates(map[string]any{"state": models.CallbackPending, "attempts": 0, "last_error": "", "available_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, "callback_retried", "callbacks", &id)
	})
}
func (r *Callbacks) Cleanup(ctx context.Context) error {
	if err := r.db.WithContext(ctx).Exec("DELETE FROM sessions WHERE expiry < NOW()").Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec("DELETE FROM login_limits WHERE window_start < NOW()-INTERVAL '1 hour'").Error
}

func (r *Callbacks) Transaction(ctx context.Context, mutate func(*Callbacks) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return mutate(NewCallbacks(tx)) })
}
