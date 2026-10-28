package models

import (
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

// CompanyTestSend records a real provider request made without an API credential.
// Its initial unknown state is conservative if the process stops after dispatch.
type CompanyTestSend struct {
	ID                uint64 `gorm:"primaryKey"`
	CompanyID         uint64
	ActorID           uint64
	Destination       string
	CorrelationID     string
	ProviderMessageID *string
	State             SubmissionState
	FailureCode       apperrors.Code
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (CompanyTestSend) TableName() string { return "company_test_sends" }
