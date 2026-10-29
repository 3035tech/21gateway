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

// MessageTrace shows persisted milestones only. A missing milestone must not be
// interpreted as a provider failure or as proof of delivery.
type MessageTrace struct {
	ID                uint64                 `json:"id"`
	SubmissionState   models.SubmissionState `json:"submissionState"`
	FailureCode       string                 `json:"failureCode"`
	DeliveryStatus    string                 `json:"deliveryStatus"`
	ProviderMessageID *string                `json:"providerMessageId,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	DispatchedAt      *time.Time             `json:"dispatchedAt,omitempty"`
	Events            []MessageTraceEvent    `json:"events"`
	Truncated         bool                   `json:"truncated"`
}

type MessageTraceEvent struct {
	Kind        string    `json:"kind"`
	ID          uint64    `json:"id"`
	Status      string    `json:"status"`
	OccurredAt  time.Time `json:"occurredAt"`
	RecordedAt  time.Time `json:"recordedAt"`
	Attempts    int       `json:"attempts,omitempty"`
	FailureCode string    `json:"failureCode,omitempty"`
}

type MessageTraceOutput struct{ Body MessageTrace }

type MessageContentState string

const (
	MessageContentAvailable   MessageContentState = "available"
	MessageContentExpired     MessageContentState = "expired"
	MessageContentUnavailable MessageContentState = "unavailable"
)

type MessageContentView struct {
	State       MessageContentState `json:"state" enum:"available,expired,unavailable"`
	MessageText string              `json:"messageText,omitempty"`
}
type MessageContentOutput struct{ Body MessageContentView }

type TestSendInput struct {
	ID   uint64 `path:"id" minimum:"1"`
	Body struct {
		Destination string `json:"destination" pattern:"^[0-9]{11}$"`
		MessageText string `json:"messageText,omitempty" maxLength:"160"`
	}
}

type TestSendResult struct {
	SubmissionState models.SubmissionState `json:"submissionState"`
	IDSMS           string                 `json:"idSms,omitempty"`
	CorrelationID   string                 `json:"correlationId"`
	FailureCode     string                 `json:"failureCode,omitempty"`
}

type CompanyTestSendInput struct {
	PublicRef string `path:"publicRef" format:"uuid"`
	Body      struct {
		Destination string `json:"destination" pattern:"^[0-9]{11}$"`
		MessageText string `json:"messageText,omitempty" maxLength:"160"`
	}
}

type TestSendOutput struct{ Body TestSendResult }

type CompanyTestHistoryInput struct {
	PublicRef string `path:"publicRef" format:"uuid"`
}

type CompanyTestHistoryItem struct {
	Destination       string                 `json:"destination"`
	CorrelationID     string                 `json:"correlationId"`
	ProviderMessageID *string                `json:"providerMessageId"`
	State             models.SubmissionState `json:"state"`
	FailureCode       string                 `json:"failureCode"`
	CreatedAt         time.Time              `json:"createdAt"`
}

type CompanyTestHistoryOutput struct{ Body []CompanyTestHistoryItem }

type ZougDiagnosticInput struct {
	PublicRef string `path:"publicRef" format:"uuid"`
	Body      struct {
		Destination string `json:"destination,omitempty"`
	}
}
type ZougDiagnosticResult struct {
	RunID   string               `json:"runId"`
	Results []ZougDiagnosticCase `json:"results"`
}
type ZougDiagnosticCase struct {
	CaseID          string              `json:"caseId"`
	RequestMethod   string              `json:"requestMethod"`
	RequestPath     string              `json:"requestPath"`
	RequestBody     string              `json:"requestBody"`
	ContentType     string              `json:"contentType"`
	Auth            string              `json:"auth"`
	HTTPStatus      int                 `json:"httpStatus"`
	ResponseBody    string              `json:"responseBody"`
	ResponseHeaders map[string][]string `json:"responseHeaders,omitempty"`
	DurationMS      int64               `json:"durationMs"`
	ErrorCode       string              `json:"errorCode,omitempty"`
}
type ZougDiagnosticOutput struct{ Body ZougDiagnosticResult }
