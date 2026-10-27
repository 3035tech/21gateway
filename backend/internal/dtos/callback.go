package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type DeliveryInput struct {
	Body struct {
		EventKey          string                `json:"eventKey" minLength:"1" maxLength:"200"`
		ProviderMessageID string                `json:"providerMessageId" format:"uuid"`
		Status            models.DeliveryStatus `json:"status" enum:"SENT,DELIVERED,NOT_DELIVERED,DUPLICATED,INVALID_DESTINATION_ADDRESS,BLACKLIST,MESSAGE_CANCELLED,INVALID_MESSAGE_FORMAT,GENERAL_ERROR"`
		OccurredAt        time.Time             `json:"occurredAt"`
	}
}
type CallbackView struct {
	ID           uint64                `json:"id"`
	AttemptID    uint64                `json:"attemptId"`
	CredentialID uint64                `json:"credentialId"`
	CompanyID    *uint64               `json:"companyId"`
	CompanyName  string                `json:"companyName"`
	URL          string                `json:"url"`
	Status       models.DeliveryStatus `json:"status"`
	State        models.CallbackState  `json:"state"`
	Attempts     int                   `json:"attempts"`
	LastError    string                `json:"lastError"`
	AvailableAt  time.Time             `json:"availableAt"`
	CreatedAt    time.Time             `json:"createdAt"`
}
type CallbackListInput struct {
	ListQuery
	CallbackState models.CallbackState `query:"callbackState" enum:"pending,processing,delivered,failed"`
}
type CallbacksOutput struct{ Body Page[CallbackView] }
