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

const sendReferenceResignal = 5 * time.Minute

// SendReferences publishes one durable SMS attempt ID at a time. The
// scheduled_sms row remains the source of truth if SQS loses an acknowledgment.
type SendReferences struct{ db *gorm.DB }

func NewSendReferences(db *gorm.DB) *SendReferences { return &SendReferences{db: db} }

func (r *SendReferences) ClaimPublication(ctx context.Context) (*models.ScheduledSMS, error) {
	var job models.ScheduledSMS
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("state='queued' AND send_reference=TRUE AND dispatch_batch_id IS NULL AND due_at<=NOW() AND available_at<=NOW() AND (publish_after IS NULL OR publish_after<=NOW())").
			Order("due_at,id").First(&job).Error
		if err != nil {
			return err
		}
		token := uuid.NewString()
		job.PublishToken = &token
		next := time.Now().UTC().Add(sendReferenceResignal)
		job.PublishAfter = &next
		return tx.Model(&job).Updates(map[string]any{"publish_token": token, "publish_after": next}).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}

func (r *SendReferences) PublicationResult(ctx context.Context, job models.ScheduledSMS, success bool, retry time.Duration) error {
	updates := map[string]any{"publish_token": nil}
	if success {
		updates["published_at"] = time.Now().UTC()
	} else {
		updates["publish_after"] = time.Now().UTC().Add(retry)
	}
	return r.db.WithContext(ctx).Model(&models.ScheduledSMS{}).Where("id=? AND publish_token=?", job.ID, job.PublishToken).Updates(updates).Error
}

// Claim one message after receiving its ID. A dispatching send with an expired
// lease has an unknown provider outcome and must never be submitted again.
func (r *SendReferences) Claim(ctx context.Context, attemptID uint64) (models.ScheduledSMS, bool, bool, error) {
	var job models.ScheduledSMS
	done := false
	ready := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("attempt_id=? AND send_reference=TRUE AND dispatch_batch_id IS NULL", attemptID).First(&job).Error; err != nil {
			return err
		}
		switch job.State {
		case models.ScheduleDone, models.ScheduleFailed, models.ScheduleUnknown:
			done = true
			return nil
		case models.ScheduleDispatching:
			if job.LeaseUntil != nil && job.LeaseUntil.After(time.Now()) {
				return nil
			}
			if err := tx.Model(&job).Updates(map[string]any{"state": models.ScheduleUnknown, "encrypted_message": "", "last_error": apperrors.CodeUnknownOutcome, "lease_token": nil, "lease_until": nil}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.SMSAttempt{}).Where("id=?", attemptID).Updates(map[string]any{"state": models.SubmissionUnknown, "failure_code": apperrors.CodeUnknownOutcome}).Error; err != nil {
				return err
			}
			done = true
			return nil
		case models.ScheduleClaimed:
			if job.LeaseUntil != nil && job.LeaseUntil.After(time.Now()) {
				return nil
			}
		case models.ScheduleQueued:
		default:
			return nil
		}
		if job.DueAt.After(time.Now()) || job.AvailableAt.After(time.Now()) {
			return nil
		}
		token := uuid.NewString()
		until := time.Now().UTC().Add(models.ScheduleLease)
		if err := tx.Model(&job).Updates(map[string]any{"state": models.ScheduleClaimed, "lease_token": token, "lease_until": until}).Error; err != nil {
			return err
		}
		job.State, job.LeaseToken, job.LeaseUntil = models.ScheduleClaimed, &token, &until
		if err := tx.Where("id=?", job.AttemptID).First(&job.Attempt).Error; err != nil {
			return err
		}
		ready = true
		return nil
	})
	return job, done, ready, err
}
