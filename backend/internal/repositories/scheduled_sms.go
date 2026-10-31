package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const scheduleClaimLockNamespace = 210017

type ScheduledSMS struct{ db *gorm.DB }

func NewScheduledSMS(db *gorm.DB) *ScheduledSMS { return &ScheduledSMS{db: db} }
func (r *ScheduledSMS) List(ctx context.Context, q dtos.ScheduledListQuery, from, to *time.Time) (dtos.Page[dtos.ScheduledView], error) {
	page := dtos.Page[dtos.ScheduledView]{Items: []dtos.ScheduledView{}, Page: q.Page, PageSize: q.PageSize}
	if err := validatePage(dtos.ListQuery{Page: q.Page, PageSize: q.PageSize, Direction: q.Direction}); err != nil {
		return page, err
	}
	db := r.db.WithContext(ctx).Table("scheduled_sms AS s").
		Joins("JOIN sms_attempts AS a ON a.id=s.attempt_id").
		Joins("JOIN api_credentials AS cred ON cred.id=s.credential_id").
		Joins("LEFT JOIN companies AS c ON c.id=a.company_id").
		Where("a.scheduled_at IS NOT NULL")
	db = scopedCompanies(db, ctx, "a.company_id")
	if q.CompanyID != 0 {
		db = db.Where("a.company_id=?", q.CompanyID)
	}
	if q.CredentialID != 0 {
		db = db.Where("s.credential_id=?", q.CredentialID)
	}
	if q.State != "" {
		db = db.Where("s.state=?", q.State)
	}
	if q.Search != "" {
		db = db.Where("a.correlation_id ILIKE ? OR a.destination ILIKE ? OR a.id::text = ?", searchTerm(q.Search), searchTerm(q.Search), q.Search)
	}
	if from != nil {
		db = db.Where("s.due_at>=?", *from)
	}
	if to != nil {
		db = db.Where("s.due_at<=?", *to)
	}
	if err := db.Count(&page.Total).Error; err != nil {
		return page, err
	}
	column := map[string]string{
		"id": "s.id", "attemptId": "s.attempt_id", "companyName": "c.name", "username": "cred.username",
		"destination": "a.destination", "correlationId": "a.correlation_id", "scheduledAt": "s.due_at",
		"state": "s.state", "submissionState": "a.state", "dispatchedAt": "a.dispatched_at", "createdAt": "s.created_at",
	}[q.Sort]
	if q.Sort == "" {
		column = "s.due_at"
	}
	if column == "" {
		return page, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	direction := string(q.Direction)
	err := db.Select("s.id, s.attempt_id, a.company_id, COALESCE(c.name,'') AS company_name, s.credential_id, cred.username, a.destination, a.correlation_id, s.due_at AS scheduled_at, a.dispatched_at, s.state, a.state AS submission_state, s.created_at").
		Order(column + " " + direction).
		Order("s.id " + direction).
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&page.Items).Error
	return page, err
}
func (r *ScheduledSMS) Enqueue(ctx context.Context, jobs []models.ScheduledSMS) ([]models.ScheduledSMS, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		attempts := make([]models.SMSAttempt, len(jobs))
		for i := range jobs {
			attempts[i] = jobs[i].Attempt
		}
		if err := tx.Create(&attempts).Error; err != nil {
			return err
		}
		for i := range jobs {
			jobs[i].Attempt = attempts[i]
			jobs[i].AttemptID = attempts[i].ID
		}
		return tx.Create(&jobs).Error
	})
	return jobs, err
}
func (r *ScheduledSMS) Claim(ctx context.Context) ([]models.ScheduledSMS, error) {
	jobs := []models.ScheduledSMS{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Never replay a dispatch whose process died after crossing the send boundary.
		if err := tx.Exec(`WITH expired AS (SELECT id FROM scheduled_sms WHERE send_reference=FALSE AND state='dispatching' AND lease_until<NOW() ORDER BY lease_until,id LIMIT ? FOR UPDATE SKIP LOCKED), marked AS (UPDATE scheduled_sms s SET state='unknown',encrypted_message='',last_error=?,lease_token=NULL,lease_until=NULL,updated_at=NOW() FROM expired WHERE s.id=expired.id RETURNING s.attempt_id) UPDATE sms_attempts a SET state='unknown',failure_code=?,updated_at=NOW() FROM marked WHERE a.id=marked.attempt_id`, models.ScheduledBatchSize, apperrors.CodeUnknownOutcome, apperrors.CodeUnknownOutcome).Error; err != nil {
			return err
		}
		var first models.ScheduledSMS
		// Literal state predicates let PostgreSQL match partial indexes in generic plans.
		eligible := `due_at<=NOW() AND available_at<=NOW() AND (state='queued' OR (state='claimed' AND lease_until<NOW()))`
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("dispatch_batch_id IS NULL AND send_reference=FALSE").Where(eligible).Where(`NOT EXISTS (SELECT 1 FROM scheduled_sms busy WHERE busy.credential_id=scheduled_sms.credential_id AND busy.send_reference=FALSE AND busy.state IN ('claimed','dispatching') AND busy.lease_until>=NOW())`).Order("available_at,due_at,id").First(&first).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		var firstAttempt models.SMSAttempt
		if err := tx.Select("provider_code").First(&firstAttempt, first.AttemptID).Error; err != nil {
			return err
		}
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?,?)", scheduleClaimLockNamespace, int32(first.CredentialID)).Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			return nil
		}
		var busy int64
		if err := tx.Model(&models.ScheduledSMS{}).Where("credential_id=? AND send_reference=FALSE AND state IN ('claimed','dispatching') AND lease_until>=NOW()", first.CredentialID).Count(&busy).Error; err != nil {
			return err
		}
		if busy != 0 {
			return nil
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("credential_id=? AND dispatch_batch_id IS NULL AND send_reference=FALSE", first.CredentialID).Where("EXISTS (SELECT 1 FROM sms_attempts a WHERE a.id=scheduled_sms.attempt_id AND a.provider_code=?)", firstAttempt.ProviderCode).Where(eligible).Order("due_at,id").Limit(models.ScheduledBatchSize).Find(&jobs).Error; err != nil {
			return err
		}
		if len(jobs) == 0 {
			return nil
		}
		ids := scheduleIDs(jobs)
		token := uuid.NewString()
		until := time.Now().UTC().Add(models.ScheduleLease)
		if err := tx.Model(&models.ScheduledSMS{}).Where("id IN ?", ids).Updates(map[string]any{"state": models.ScheduleClaimed, "lease_token": token, "lease_until": until}).Error; err != nil {
			return err
		}
		for i := range jobs {
			jobs[i].State = models.ScheduleClaimed
			jobs[i].LeaseToken = &token
			jobs[i].LeaseUntil = &until
		}
		return hydrateScheduledAttempts(tx, jobs)
	})
	return jobs, err
}
func (r *ScheduledSMS) Credential(ctx context.Context, id uint64) (models.APICredential, error) {
	var value models.APICredential
	err := r.db.WithContext(ctx).Model(&models.APICredential{}).Select("api_credentials.*, companies.active AS company_active, companies.provider_token_encrypted AS company_provider_token_encrypted, companies.provider_token_legacy_fallback AS company_provider_token_legacy_fallback, sms_route.provider_code AS company_sms_provider").Joins("LEFT JOIN companies ON companies.id=api_credentials.company_id").Joins("LEFT JOIN company_provider_routes AS sms_route ON sms_route.company_id=companies.id AND sms_route.channel=?", models.ChannelSMS).Where("api_credentials.id=? AND api_credentials.archived_at IS NULL AND (companies.id IS NULL OR companies.archived_at IS NULL)", id).First(&value).Error
	return value, err
}
func (r *ScheduledSMS) Begin(ctx context.Context, jobs []models.ScheduledSMS) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ScheduledSMS{}).Where("id IN ? AND state=? AND lease_token=? AND lease_until>NOW()", scheduleIDs(jobs), models.ScheduleClaimed, jobs[0].LeaseToken).Updates(map[string]any{"state": models.ScheduleDispatching, "lease_until": time.Now().UTC().Add(models.ScheduleLease)})
		if err := scheduleUpdated(result, len(jobs)); err != nil {
			return err
		}
		ids := make([]uint64, len(jobs))
		for i := range jobs {
			ids[i] = jobs[i].AttemptID
		}
		return tx.Model(&models.SMSAttempt{}).Where("id IN ?", ids).Update("dispatched_at", time.Now().UTC()).Error
	})
}
func (r *ScheduledSMS) Release(ctx context.Context, jobs []models.ScheduledSMS) error {
	delay := time.Second
	if jobs[0].SendReference {
		delay = 30 * time.Second
	}
	resume := time.Now().UTC().Add(delay)
	result := r.db.WithContext(ctx).Model(&models.ScheduledSMS{}).Where("id IN ? AND state=? AND lease_token=?", scheduleIDs(jobs), models.ScheduleClaimed, jobs[0].LeaseToken).Updates(map[string]any{"state": models.ScheduleQueued, "available_at": resume, "publish_after": resume, "lease_token": nil, "lease_until": nil})
	return scheduleUpdated(result, len(jobs))
}
func (r *ScheduledSMS) Reject(ctx context.Context, jobs []models.ScheduledSMS, code apperrors.Code) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.ScheduledSMS{}).Where("id IN ? AND state=? AND lease_token=?", scheduleIDs(jobs), models.ScheduleClaimed, jobs[0].LeaseToken).Updates(map[string]any{"state": models.ScheduleFailed, "last_error": code, "encrypted_message": "", "lease_token": nil, "lease_until": nil})
		if err := scheduleUpdated(result, len(jobs)); err != nil {
			return err
		}
		ids := make([]uint64, len(jobs))
		for i := range jobs {
			ids[i] = jobs[i].AttemptID
		}
		return tx.Model(&models.SMSAttempt{}).Where("id IN ?", ids).Updates(map[string]any{"state": models.SubmissionRejected, "failure_code": code}).Error
	})
}
func (r *ScheduledSMS) Finish(ctx context.Context, jobs []models.ScheduledSMS, attempts []models.SMSAttempt) error {
	type outcome struct {
		ID    uint64               `json:"id"`
		State models.ScheduleState `json:"state"`
		Code  apperrors.Code       `json:"code"`
	}
	if len(attempts) != len(jobs) {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	values := make([]outcome, len(jobs))
	for i, a := range attempts {
		if a.ID != jobs[i].AttemptID || a.CredentialID != jobs[i].CredentialID {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		state := models.ScheduleFailed
		if a.State == models.SubmissionAccepted {
			state = models.ScheduleDone
		} else if a.State == models.SubmissionUnknown {
			state = models.ScheduleUnknown
		}
		values[i] = outcome{ID: jobs[i].ID, State: state, Code: a.FailureCode}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`UPDATE scheduled_sms s SET state=o.state,last_error=o.code,encrypted_message='',lease_token=NULL,lease_until=NULL,updated_at=NOW() FROM jsonb_to_recordset(?::jsonb) AS o(id BIGINT,state TEXT,code TEXT) WHERE s.id=o.id AND s.state=? AND s.lease_token=?`, string(data), models.ScheduleDispatching, jobs[0].LeaseToken)
		if err := scheduleUpdated(result, len(jobs)); err != nil {
			return err
		}
		return NewSMSAttempts(tx).CompleteBatch(ctx, attempts)
	})
}
func scheduleIDs(jobs []models.ScheduledSMS) []uint64 {
	ids := make([]uint64, len(jobs))
	for i := range jobs {
		ids[i] = jobs[i].ID
	}
	return ids
}
func scheduleUpdated(result *gorm.DB, want int) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(want) {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	return nil
}

func hydrateScheduledAttempts(tx *gorm.DB, jobs []models.ScheduledSMS) error {
	ids := make([]uint64, len(jobs))
	for i := range jobs {
		ids[i] = jobs[i].AttemptID
	}
	var attempts []models.SMSAttempt
	if err := tx.Where("id IN ?", ids).Find(&attempts).Error; err != nil {
		return err
	}
	byID := make(map[uint64]models.SMSAttempt, len(attempts))
	for _, a := range attempts {
		byID[a.ID] = a
	}
	for i := range jobs {
		a, ok := byID[jobs[i].AttemptID]
		if !ok || a.CredentialID != jobs[i].CredentialID {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		jobs[i].Attempt = a
	}
	return nil
}
