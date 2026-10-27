package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"time"
)

const summaryQueryTimeout = 5 * time.Second

type Summary struct{ *store }

func NewSummary(db *gorm.DB) *Summary { return &Summary{store: &store{db: db}} }
func (r *Summary) Summary(ctx context.Context) (dtos.Summary, error) {
	var result dtos.Summary
	ctx, cancel := context.WithTimeout(ctx, summaryQueryTimeout)
	defer cancel()
	db := r.db.WithContext(ctx)
	if err := db.Model(&models.Company{}).Where("active AND archived_at IS NULL").Count(&result.Companies).Error; err != nil {
		return result, err
	}
	if err := db.Model(&models.APICredential{}).Where("active AND archived_at IS NULL").Count(&result.ActiveCredentials).Error; err != nil {
		return result, err
	}
	// One pass over messages, rather than three full historical counts.
	var messages struct {
		Accepted int64
		Unknown  int64
		Rejected int64
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state=?) AS accepted,COUNT(*) FILTER (WHERE state=?) AS unknown,COUNT(*) FILTER (WHERE state=?) AS rejected FROM sms_attempts`, models.SubmissionAccepted, models.SubmissionUnknown, models.SubmissionRejected).Scan(&messages).Error; err != nil {
		return result, err
	}
	result.Accepted = messages.Accepted
	result.Unknown = messages.Unknown
	result.Rejected = messages.Rejected
	var callbacks struct {
		Pending int64
		Failed  int64
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state IN (?,?)) AS pending,COUNT(*) FILTER (WHERE state=?) AS failed FROM callback_jobs`, models.CallbackPending, models.CallbackProcessing, models.CallbackFailed).Scan(&callbacks).Error; err != nil {
		return result, err
	}
	result.PendingCallbacks = callbacks.Pending
	result.FailedCallbacks = callbacks.Failed
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state IN (?,?)) AS pending,COUNT(*) FILTER (WHERE state=?) AS failed FROM mo_callback_jobs`, models.CallbackPending, models.CallbackProcessing, models.CallbackFailed).Scan(&callbacks).Error; err != nil {
		return result, err
	}
	result.PendingMOCallbacks = callbacks.Pending
	result.FailedMOCallbacks = callbacks.Failed
	var scheduled struct {
		Queued      int64
		Due         int64
		Next24Hours int64
		Active      int64
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state='queued') AS queued,COUNT(*) FILTER (WHERE state='queued' AND due_at<=NOW()) AS due,COUNT(*) FILTER (WHERE state='queued' AND due_at>NOW() AND due_at<=NOW()+INTERVAL '24 hours') AS next24_hours,COUNT(*) FILTER (WHERE state IN ('claimed','dispatching')) AS active FROM scheduled_sms WHERE state IN ('queued','claimed','dispatching')`).Scan(&scheduled).Error; err != nil {
		return result, err
	}
	result.ScheduledQueued = scheduled.Queued
	result.ScheduledDue = scheduled.Due
	result.ScheduledNext24h = scheduled.Next24Hours
	result.ScheduledActive = scheduled.Active
	return result, nil
}
