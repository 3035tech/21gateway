package repositories

import (
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
	"sort"
	"time"
)

type SMSAttempts struct{ db *gorm.DB }

func NewSMSAttempts(db *gorm.DB) *SMSAttempts { return &SMSAttempts{db: db} }
func (r *SMSAttempts) Submission(ctx context.Context, credentialID, sendID uint64) (models.SMSAttempt, error) {
	var attempt models.SMSAttempt
	err := r.db.WithContext(ctx).Model(&models.SMSAttempt{}).
		Select("id,state,failure_code,delivery_status,created_at,dispatched_at").
		Where("id=? AND credential_id=?", sendID, credentialID).First(&attempt).Error
	return attempt, err
}
func (r *SMSAttempts) Create(ctx context.Context, attempt *models.SMSAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}
func (r *SMSAttempts) CreateCompanyTest(ctx context.Context, send *models.CompanyTestSend) error {
	return r.db.WithContext(ctx).Create(send).Error
}
func (r *SMSAttempts) CompleteCompanyTest(ctx context.Context, send models.CompanyTestSend) error {
	result := r.db.WithContext(ctx).Model(&models.CompanyTestSend{}).Where("id = ? AND correlation_id = ?", send.ID, send.CorrelationID).Updates(map[string]any{
		"state": send.State, "provider_message_id": send.ProviderMessageID, "failure_code": send.FailureCode, "updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *SMSAttempts) CompanyTests(ctx context.Context, companyID uint64) ([]dtos.CompanyTestHistoryItem, error) {
	items := []dtos.CompanyTestHistoryItem{}
	err := scopedCompanies(r.db.WithContext(ctx).Model(&models.CompanyTestSend{}), ctx, "company_id").
		Where("company_id = ?", companyID).Select("destination, correlation_id, provider_message_id, state, failure_code, created_at").
		Order("created_at DESC, id DESC").Limit(20).Scan(&items).Error
	return items, err
}
func (r *SMSAttempts) Complete(ctx context.Context, attempt models.SMSAttempt) error {
	result := r.db.WithContext(ctx).Model(&models.SMSAttempt{}).
		Where("id = ? AND credential_id = ?", attempt.ID, attempt.CredentialID).
		Updates(map[string]any{"encoding": attempt.Encoding, "state": attempt.State, "provider_message_id": attempt.ProviderMessageID, "failure_code": attempt.FailureCode})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (r *SMSAttempts) Messages(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.MessageView], error) {
	result := dtos.Page[dtos.MessageView]{Items: []dtos.MessageView{}, Page: q.Page, PageSize: q.PageSize}
	db := r.db.WithContext(ctx).Table("sms_attempts")
	db = scopedCompanies(db, ctx, "sms_attempts.company_id")
	if q.Search != "" {
		db = db.Where("sms_attempts.correlation_id ILIKE ? OR sms_attempts.destination ILIKE ? OR sms_attempts.id::text = ?", searchTerm(q.Search), searchTerm(q.Search), q.Search)
	}
	if q.CompanyID != 0 {
		db = db.Where("sms_attempts.company_id = ?", q.CompanyID)
	}
	if q.State != "" {
		db = db.Where("sms_attempts.state = ?", q.State)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	if err := validatePage(q); err != nil {
		return result, err
	}
	column := map[string]string{"id": "sms_attempts.id", "companyName": "companies.name", "createdAt": "sms_attempts.created_at", "state": "sms_attempts.state", "destination": "sms_attempts.destination", "correlationId": "sms_attempts.correlation_id", "deliveryStatus": "sms_attempts.delivery_status", "failureCode": "sms_attempts.failure_code"}[q.Sort]
	if q.Sort == "" {
		column = "sms_attempts.created_at"
	}
	if column == "" {
		return result, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	direction := string(q.Direction)
	err := db.Joins("LEFT JOIN companies ON companies.id=sms_attempts.company_id").Select("sms_attempts.id,sms_attempts.credential_id,sms_attempts.company_id,COALESCE(companies.name,'') AS company_name,sms_attempts.destination,sms_attempts.correlation_id,sms_attempts.state,sms_attempts.failure_code,sms_attempts.delivery_status,sms_attempts.created_at").Order(column + " " + direction).Order("sms_attempts.id " + direction).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
}
func (r *SMSAttempts) MessageContent(ctx context.Context, id uint64) (models.SMSAttempt, error) {
	var attempt models.SMSAttempt
	err := scopedCompanies(r.db.WithContext(ctx).Model(&models.SMSAttempt{}), ctx, "company_id").Select("id,credential_id,content_ref,encrypted_message,content_purged_at,created_at,scheduled_at,dispatched_at").Where("id = ?", id).First(&attempt).Error
	return attempt, err
}

const messageTraceLimit = 100

func (r *SMSAttempts) MessageTrace(ctx context.Context, id uint64) (dtos.MessageTrace, error) {
	var attempt models.SMSAttempt
	err := scopedCompanies(r.db.WithContext(ctx).Model(&models.SMSAttempt{}), ctx, "company_id").
		Select("id,state,failure_code,delivery_status,provider_message_id,created_at,dispatched_at").
		Where("id = ?", id).First(&attempt).Error
	if err != nil {
		return dtos.MessageTrace{}, err
	}
	trace := dtos.MessageTrace{
		ID: attempt.ID, SubmissionState: attempt.State, FailureCode: string(attempt.FailureCode),
		DeliveryStatus: attempt.DeliveryStatus, ProviderMessageID: attempt.ProviderMessageID,
		CreatedAt: attempt.CreatedAt, DispatchedAt: attempt.DispatchedAt,
		Events: []dtos.MessageTraceEvent{},
	}
	var deliveries []models.DeliveryEvent
	if err := r.db.WithContext(ctx).Model(&models.DeliveryEvent{}).
		Select("id,event_key,status,occurred_at,created_at").Where("attempt_id = ?", id).
		Order("occurred_at,id").Limit(messageTraceLimit + 1).Find(&deliveries).Error; err != nil {
		return dtos.MessageTrace{}, err
	}
	for _, event := range deliveries {
		kind := "delivery"
		if len(event.EventKey) >= 5 && event.EventKey[:5] == "zoug:" {
			kind = "zoug_delivery"
		}
		trace.Events = append(trace.Events, dtos.MessageTraceEvent{Kind: kind, ID: event.ID, Status: string(event.Status), OccurredAt: event.OccurredAt, RecordedAt: event.CreatedAt})
	}
	var mos []models.MOEvent
	if err := r.db.WithContext(ctx).Model(&models.MOEvent{}).
		Select("id,received_at,created_at").Where("attempt_id = ?", id).
		Order("received_at,id").Limit(messageTraceLimit + 1).Find(&mos).Error; err != nil {
		return dtos.MessageTrace{}, err
	}
	for _, event := range mos {
		trace.Events = append(trace.Events, dtos.MessageTraceEvent{Kind: "mo", ID: event.ID, Status: "received", OccurredAt: event.ReceivedAt, RecordedAt: event.CreatedAt})
	}
	var dlrJobs []models.CallbackJob
	if err := r.db.WithContext(ctx).Model(&models.CallbackJob{}).
		Select("id,state,attempts,last_error,created_at,updated_at").Where("attempt_id = ?", id).
		Order("created_at,id").Limit(messageTraceLimit + 1).Find(&dlrJobs).Error; err != nil {
		return dtos.MessageTrace{}, err
	}
	for _, job := range dlrJobs {
		trace.Events = append(trace.Events, dtos.MessageTraceEvent{Kind: "client_dlr", ID: job.ID, Status: string(job.State), OccurredAt: job.UpdatedAt, RecordedAt: job.CreatedAt, Attempts: job.Attempts, FailureCode: job.LastError})
	}
	var moJobs []models.MOCallbackJob
	if err := r.db.WithContext(ctx).Model(&models.MOCallbackJob{}).
		Select("id,state,attempts,last_error,created_at,updated_at").Where("attempt_id = ?", id).
		Order("created_at,id").Limit(messageTraceLimit + 1).Find(&moJobs).Error; err != nil {
		return dtos.MessageTrace{}, err
	}
	for _, job := range moJobs {
		trace.Events = append(trace.Events, dtos.MessageTraceEvent{Kind: "client_mo", ID: job.ID, Status: string(job.State), OccurredAt: job.UpdatedAt, RecordedAt: job.CreatedAt, Attempts: job.Attempts, FailureCode: job.LastError})
	}
	sort.SliceStable(trace.Events, func(i, j int) bool { return trace.Events[i].OccurredAt.Before(trace.Events[j].OccurredAt) })
	if len(trace.Events) > messageTraceLimit {
		trace.Events = trace.Events[:messageTraceLimit]
		trace.Truncated = true
	}
	return trace, nil
}

func (r *SMSAttempts) PurgeContent(ctx context.Context, before time.Time, limit int) (int64, error) {
	result := r.db.WithContext(ctx).Exec(`WITH expired AS (SELECT id FROM sms_attempts WHERE encrypted_message <> '' AND COALESCE(dispatched_at,scheduled_at,created_at) < ? ORDER BY COALESCE(dispatched_at,scheduled_at,created_at),id LIMIT ? FOR UPDATE SKIP LOCKED) UPDATE sms_attempts AS a SET encrypted_message='',content_ref=NULL,content_purged_at=NOW() FROM expired WHERE a.id=expired.id`, before, limit)
	return result.RowsAffected, result.Error
}

func (r *SMSAttempts) CreateBatch(ctx context.Context, attempts []*models.SMSAttempt) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return tx.Create(attempts).Error })
}

func (r *SMSAttempts) CompleteBatch(ctx context.Context, attempts []models.SMSAttempt) error {
	type outcome struct {
		ID                uint64                 `json:"id"`
		CredentialID      uint64                 `json:"credential_id"`
		State             models.SubmissionState `json:"state"`
		ProviderMessageID *string                `json:"provider_message_id"`
		FailureCode       string                 `json:"failure_code"`
		Encoding          models.SMSEncoding     `json:"encoding"`
	}
	values := make([]outcome, len(attempts))
	for i, v := range attempts {
		values[i] = outcome{ID: v.ID, CredentialID: v.CredentialID, State: v.State, ProviderMessageID: v.ProviderMessageID, FailureCode: string(v.FailureCode), Encoding: v.Encoding}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Exec(`UPDATE sms_attempts AS sms SET encoding=outcome.encoding,state=outcome.state,provider_message_id=outcome.provider_message_id,failure_code=outcome.failure_code,updated_at=NOW() FROM jsonb_to_recordset(?::jsonb) AS outcome(id BIGINT,credential_id BIGINT,state TEXT,provider_message_id UUID,failure_code TEXT,encoding TEXT) WHERE sms.id=outcome.id AND sms.credential_id=outcome.credential_id`, string(data))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(values)) {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
