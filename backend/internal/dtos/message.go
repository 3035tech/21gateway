package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type MessageView struct {
	ID             uint64                 `json:"id"`
	CredentialID   uint64                 `json:"credentialId"`
	CompanyID      *uint64                `json:"companyId"`
	CompanyName    string                 `json:"companyName"`
	Destination    string                 `json:"destination"`
	CorrelationID  string                 `json:"correlationId"`
	State          models.SubmissionState `json:"state"`
	FailureCode    string                 `json:"failureCode"`
	DeliveryStatus string                 `json:"deliveryStatus"`
	CreatedAt      time.Time              `json:"createdAt"`
}
type MessagesOutput struct{ Body Page[MessageView] }

type TestSendInput struct {
	ID   uint64 `path:"id" minimum:"1"`
	Body struct {
		Destination string `json:"destination" pattern:"^[0-9]{11}$"`
	}
}

type TestSendResult struct {
	SubmissionState models.SubmissionState `json:"submissionState"`
	IDSMS           string                 `json:"idSms,omitempty"`
	CorrelationID   string                 `json:"correlationId"`
	FailureCode     string                 `json:"failureCode,omitempty"`
}

type TestSendOutput struct{ Body TestSendResult }
