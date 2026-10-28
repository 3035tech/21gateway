package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"gorm.io/gorm"
	"time"
)

const summaryQueryTimeout = 5 * time.Second

type Summary struct{ *store }

func NewSummary(db *gorm.DB) *Summary { return &Summary{store: &store{db: db}} }
func (r *Summary) Summary(ctx context.Context, companyID ...uint64) (dtos.Summary, error) {
	var result dtos.Summary
	ctx, cancel := context.WithTimeout(ctx, summaryQueryTimeout)
	defer cancel()
	db := r.db.WithContext(ctx)
	if len(companyID) > 0 && companyID[0] != 0 {
		if !security.AllowsCompany(ctx, companyID[0]) {
			return result, gorm.ErrRecordNotFound
		}
		return scopedSummary(db, []uint64{companyID[0]})
	}
	if scope := security.Scope(ctx); scope.Restricted {
		return scopedSummary(db, scope.IDs)
	}
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
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state=?) AS accepted,COUNT(*) FILTER (WHERE state=?) AS unknown,COUNT(*) FILTER (WHERE state=?) AS rejected FROM company_test_sends`, models.SubmissionAccepted, models.SubmissionUnknown, models.SubmissionRejected).Scan(&messages).Error; err != nil {
		return result, err
	}
	result.Accepted += messages.Accepted
	result.Unknown += messages.Unknown
	result.Rejected += messages.Rejected
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
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE s.state='queued') AS queued,COUNT(*) FILTER (WHERE s.state='queued' AND s.due_at<=NOW()) AS due,COUNT(*) FILTER (WHERE s.state='queued' AND s.due_at>NOW() AND s.due_at<=NOW()+INTERVAL '24 hours') AS next24_hours,COUNT(*) FILTER (WHERE s.state IN ('claimed','dispatching')) AS active FROM scheduled_sms s JOIN sms_attempts a ON a.id=s.attempt_id WHERE a.scheduled_at IS NOT NULL AND s.state IN ('queued','claimed','dispatching')`).Scan(&scheduled).Error; err != nil {
		return result, err
	}
	result.ScheduledQueued = scheduled.Queued
	result.ScheduledDue = scheduled.Due
	result.ScheduledNext24h = scheduled.Next24Hours
	result.ScheduledActive = scheduled.Active
	return result, nil
}

func scopedSummary(db *gorm.DB, companyIDs []uint64) (dtos.Summary, error) {
	var result dtos.Summary
	if err := db.Table("companies").Where("id IN ? AND active AND archived_at IS NULL", companyIDs).Count(&result.Companies).Error; err != nil {
		return result, err
	}
	if err := db.Table("api_credentials").Where("company_id IN ? AND active AND archived_at IS NULL", companyIDs).Count(&result.ActiveCredentials).Error; err != nil {
		return result, err
	}
	var messages struct{ Accepted, Unknown, Rejected int64 }
	if err := db.Table("sms_attempts").Where("company_id IN ?", companyIDs).Select("COUNT(*) FILTER (WHERE state=?) AS accepted, COUNT(*) FILTER (WHERE state=?) AS unknown, COUNT(*) FILTER (WHERE state=?) AS rejected", models.SubmissionAccepted, models.SubmissionUnknown, models.SubmissionRejected).Scan(&messages).Error; err != nil {
		return result, err
	}
	result.Accepted, result.Unknown, result.Rejected = messages.Accepted, messages.Unknown, messages.Rejected
	if err := db.Table("company_test_sends").Where("company_id IN ?", companyIDs).Select("COUNT(*) FILTER (WHERE state=?) AS accepted, COUNT(*) FILTER (WHERE state=?) AS unknown, COUNT(*) FILTER (WHERE state=?) AS rejected", models.SubmissionAccepted, models.SubmissionUnknown, models.SubmissionRejected).Scan(&messages).Error; err != nil {
		return result, err
	}
	result.Accepted += messages.Accepted
	result.Unknown += messages.Unknown
	result.Rejected += messages.Rejected
	var callbacks struct{ Pending, Failed int64 }
	if err := db.Table("callback_jobs AS j").Joins("JOIN sms_attempts AS a ON a.id=j.attempt_id").Where("a.company_id IN ?", companyIDs).Select("COUNT(*) FILTER (WHERE j.state IN (?,?)) AS pending, COUNT(*) FILTER (WHERE j.state=?) AS failed", models.CallbackPending, models.CallbackProcessing, models.CallbackFailed).Scan(&callbacks).Error; err != nil {
		return result, err
	}
	result.PendingCallbacks, result.FailedCallbacks = callbacks.Pending, callbacks.Failed
	callbacks = struct{ Pending, Failed int64 }{}
	if err := db.Table("mo_callback_jobs AS j").Joins("JOIN sms_attempts AS a ON a.id=j.attempt_id").Where("a.company_id IN ?", companyIDs).Select("COUNT(*) FILTER (WHERE j.state IN (?,?)) AS pending, COUNT(*) FILTER (WHERE j.state=?) AS failed", models.CallbackPending, models.CallbackProcessing, models.CallbackFailed).Scan(&callbacks).Error; err != nil {
		return result, err
	}
	result.PendingMOCallbacks, result.FailedMOCallbacks = callbacks.Pending, callbacks.Failed
	var scheduled struct{ Queued, Due, Next24Hours, Active int64 }
	if err := db.Table("scheduled_sms AS s").Joins("JOIN sms_attempts AS a ON a.id=s.attempt_id").Where("a.company_id IN ? AND a.scheduled_at IS NOT NULL AND s.state IN ('queued','claimed','dispatching')", companyIDs).Select("COUNT(*) FILTER (WHERE s.state='queued') AS queued, COUNT(*) FILTER (WHERE s.state='queued' AND s.due_at<=NOW()) AS due, COUNT(*) FILTER (WHERE s.state='queued' AND s.due_at>NOW() AND s.due_at<=NOW()+INTERVAL '24 hours') AS next24_hours, COUNT(*) FILTER (WHERE s.state IN ('claimed','dispatching')) AS active").Scan(&scheduled).Error; err != nil {
		return result, err
	}
	result.ScheduledQueued, result.ScheduledDue, result.ScheduledNext24h, result.ScheduledActive = scheduled.Queued, scheduled.Due, scheduled.Next24Hours, scheduled.Active
	return result, nil
}
