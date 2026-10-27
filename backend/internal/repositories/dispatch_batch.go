package repositories

import (
	"context"
	"errors"
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const dispatchResignalInterval = 5 * time.Minute
const dispatchReferenceLease = 30 * time.Second

type DispatchBatches struct{ db *gorm.DB }

func NewDispatchBatches(db *gorm.DB) *DispatchBatches { return &DispatchBatches{db: db} }
func (r *DispatchBatches) Prepare(ctx context.Context) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidate struct{ ID uint64 }
		// One open batch per account, oldest-served account first. Do not scan every
		// due SMS belonging to accounts that already have an outstanding batch.
		if err := tx.Raw(`SELECT c.id FROM api_credentials c WHERE NOT EXISTS (SELECT 1 FROM sms_dispatch_batches b WHERE b.credential_id=c.id AND b.completed_at IS NULL) AND EXISTS (SELECT 1 FROM scheduled_sms s WHERE s.credential_id=c.id AND s.dispatch_batch_id IS NULL AND s.due_at<=NOW() AND s.available_at<=NOW() AND (s.state='queued' OR (s.state='claimed' AND s.lease_until<NOW()))) ORDER BY COALESCE((SELECT MAX(h.created_at) FROM sms_dispatch_batches h WHERE h.credential_id=c.id),'epoch'::timestamptz),c.id LIMIT 1 FOR UPDATE OF c SKIP LOCKED`).Scan(&candidate).Error; err != nil {
			return err
		}
		if candidate.ID == 0 {
			return nil
		}
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?,?)", scheduleClaimLockNamespace, int32(candidate.ID)).Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			return nil
		}
		var active int64
		if err := tx.Raw(`SELECT COUNT(*) FROM api_credentials c WHERE EXISTS (SELECT 1 FROM scheduled_sms s WHERE s.credential_id=c.id AND s.due_at<=NOW() AND s.state IN ('queued','claimed','dispatching'))`).Scan(&active).Error; err != nil {
			return err
		}
		batchSize := models.ScheduledBatchSize
		if active > 1 {
			batchSize = models.ScheduledBatchSize / int(active)
			if batchSize < 1 {
				batchSize = 1
			}
		}
		var jobs []models.ScheduledSMS
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("credential_id=? AND dispatch_batch_id IS NULL AND due_at<=NOW() AND available_at<=NOW() AND (state='queued' OR (state='claimed' AND lease_until<NOW()))", candidate.ID).Order("due_at,id").Limit(batchSize).Find(&jobs).Error; err != nil {
			return err
		}
		if len(jobs) == 0 {
			return nil
		}
		batch := models.DispatchBatch{ID: uuid.NewString(), CredentialID: candidate.ID, AvailableAt: time.Now().UTC()}
		if err := tx.Create(&batch).Error; err != nil {
			return err
		}
		result := tx.Model(&models.ScheduledSMS{}).Where("id IN ? AND dispatch_batch_id IS NULL", scheduleIDs(jobs)).Updates(map[string]any{"state": models.ScheduleClaimed, "dispatch_batch_id": batch.ID, "lease_token": nil, "lease_until": nil})
		return scheduleUpdated(result, len(jobs))
	})
}
func (r *DispatchBatches) ClaimPublication(ctx context.Context) (*models.DispatchBatch, error) {
	var batch models.DispatchBatch
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("completed_at IS NULL AND available_at<=NOW() AND (lease_until IS NULL OR lease_until<NOW())").Order("available_at,id").First(&batch).Error
		if err != nil {
			return err
		}
		token := uuid.NewString()
		batch.LeaseToken = &token
		until := time.Now().UTC().Add(dispatchReferenceLease)
		batch.LeaseUntil = &until
		if batch.PublishAttempts < 1000000 {
			batch.PublishAttempts++
		}
		return tx.Save(&batch).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &batch, err
}
func (r *DispatchBatches) PublicationResult(ctx context.Context, batch models.DispatchBatch, success bool, delay time.Duration) error {
	updates := map[string]any{"available_at": time.Now().UTC().Add(delay), "lease_token": nil, "lease_until": nil}
	if success {
		updates["published_at"] = time.Now().UTC()
		updates["available_at"] = time.Now().UTC().Add(dispatchResignalInterval)
	}
	result := r.db.WithContext(ctx).Model(&models.DispatchBatch{}).Where("id=? AND lease_token=? AND completed_at IS NULL", batch.ID, batch.LeaseToken).Updates(updates)
	// Completion can race the publisher acknowledgment: it is already safe to stop.
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var completed int64
	if err := r.db.WithContext(ctx).Model(&models.DispatchBatch{}).Where("id=? AND completed_at IS NOT NULL", batch.ID).Count(&completed).Error; err != nil {
		return err
	}
	if completed == 1 {
		return nil
	}
	return apperrors.Wrap(apperrors.CodeConflict, nil)
}
func (r *DispatchBatches) ClaimBatch(ctx context.Context, id string) ([]models.ScheduledSMS, bool, error) {
	jobs := []models.ScheduledSMS{}
	done := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch models.DispatchBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&batch, "id=?", id).Error; err != nil {
			return err
		}
		if batch.CompletedAt != nil {
			done = true
			return nil
		}
		// Recover only this bounded batch. A repeated SQS message must not replay an
		// expired network attempt, even if the provider response was lost.
		if err := tx.Exec(`WITH marked AS (UPDATE scheduled_sms SET state='unknown',encrypted_message='',last_error=?,lease_token=NULL,lease_until=NULL,updated_at=NOW() WHERE dispatch_batch_id=? AND state='dispatching' AND lease_until<NOW() RETURNING attempt_id) UPDATE sms_attempts a SET state='unknown',failure_code=?,updated_at=NOW() FROM marked WHERE a.id=marked.attempt_id`, apperrors.CodeUnknownOutcome, id, apperrors.CodeUnknownOutcome).Error; err != nil {
			return err
		}
		var all []models.ScheduledSMS
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("dispatch_batch_id=?", id).Order("id").Limit(models.ScheduledBatchSize + 1).Find(&all).Error; err != nil {
			return err
		}
		if len(all) == 0 || len(all) > models.ScheduledBatchSize {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		var now time.Time
		if err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		for _, job := range all {
			if job.CredentialID != batch.CredentialID {
				return apperrors.Wrap(apperrors.CodeConflict, nil)
			}
			switch job.State {
			case models.ScheduleDone, models.ScheduleFailed, models.ScheduleUnknown:
				continue
			case models.ScheduleDispatching:
				jobs = nil
				return nil
			case models.ScheduleQueued:
				if job.AvailableAt.After(now) {
					jobs = nil
					return nil
				}
			case models.ScheduleClaimed:
				if job.LeaseUntil != nil && job.LeaseUntil.After(now) {
					jobs = nil
					return nil
				}
			default:
				return apperrors.Wrap(apperrors.CodeConflict, nil)
			}
			jobs = append(jobs, job)
		}
		if len(jobs) == 0 {
			done = true
			return tx.Model(&batch).Update("completed_at", now).Error
		}
		token := uuid.NewString()
		until := now.Add(models.ScheduleLease)
		result := tx.Model(&models.ScheduledSMS{}).Where("id IN ?", scheduleIDs(jobs)).Updates(map[string]any{"state": models.ScheduleClaimed, "lease_token": token, "lease_until": until})
		if err := scheduleUpdated(result, len(jobs)); err != nil {
			return err
		}
		for i := range jobs {
			jobs[i].State = models.ScheduleClaimed
			jobs[i].LeaseToken = &token
			jobs[i].LeaseUntil = &until
		}
		return hydrateScheduledAttempts(tx, jobs)
	})
	return jobs, done, err
}
func (r *DispatchBatches) Complete(ctx context.Context, id string) (bool, error) {
	result := r.db.WithContext(ctx).Exec(`UPDATE sms_dispatch_batches b SET completed_at=COALESCE(completed_at,NOW()),updated_at=NOW() WHERE b.id=? AND EXISTS (SELECT 1 FROM scheduled_sms s WHERE s.dispatch_batch_id=b.id) AND NOT EXISTS (SELECT 1 FROM scheduled_sms s WHERE s.dispatch_batch_id=b.id AND s.state IN ('queued','claimed','dispatching'))`, id)
	return result.RowsAffected == 1, result.Error
}
