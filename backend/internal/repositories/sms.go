package repositories

import (
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
)

type SMSAttempts struct{ db *gorm.DB }

func NewSMSAttempts(db *gorm.DB) *SMSAttempts { return &SMSAttempts{db: db} }
func (r *SMSAttempts) Create(ctx context.Context, attempt *models.SMSAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
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
	if q.Search != "" {
		db = db.Where("sms_attempts.correlation_id ILIKE ? OR sms_attempts.destination ILIKE ?", searchTerm(q.Search), searchTerm(q.Search))
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
	err := db.Joins("LEFT JOIN companies ON companies.id=sms_attempts.company_id").Select("sms_attempts.*, COALESCE(companies.name,'') AS company_name").Order(column + " " + direction).Order("sms_attempts.id " + direction).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
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
