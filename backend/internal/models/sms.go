package models

import (
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

type SMSEncoding string

const (
	SMSEncodingGSM7 SMSEncoding = "0"
	SMSEncodingUCS2 SMSEncoding = "8"
)

type SubmissionState string

const (
	SubmissionPending  SubmissionState = "pending"
	SubmissionAccepted SubmissionState = "accepted"
	SubmissionRejected SubmissionState = "rejected"
	SubmissionUnknown  SubmissionState = "unknown"
)

type SMSAttempt struct {
	ScheduledAt       *time.Time
	DispatchedAt      *time.Time
	CompanyID         *uint64
	ID                uint64 `gorm:"primaryKey"`
	CredentialID      uint64
	Destination       string
	CorrelationID     string
	Encoding          SMSEncoding
	ProviderMessageID *string
	DeliveryStatus    string
	DeliveryUpdatedAt *time.Time
	State             SubmissionState
	FailureCode       apperrors.Code
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ContentRef        *string
	EncryptedMessage  string
	ContentPurgedAt   *time.Time
}

func (SMSAttempt) TableName() string { return "sms_attempts" }
