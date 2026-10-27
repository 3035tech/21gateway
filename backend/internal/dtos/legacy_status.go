package dtos

import "github.com/21mobile/sms-gateway/internal/models"

const (
	LegacyStatusPageSize                   = 100
	LegacyCodeNoRecords  LegacyResultCode  = 6
	LegacyQueryOK        LegacyMessageCode = "REQUIRED_OK"
	// The manual does not name the empty-result message; this is provisional.
	LegacyQueryEmpty LegacyMessageCode = "NO_RECORDS"
)

type LegacyStatusItem struct {
	CorrelationID string                `json:"correlationId"`
	Destination   string                `json:"destination"`
	SentDate      string                `json:"sentDate"`
	UnixSentDate  string                `json:"unixSentDate"`
	Status        models.DeliveryStatus `json:"status"`
}
type LegacyStatusBody struct {
	Status     LegacyResultStatus `json:"status"`
	Code       LegacyResultCode   `json:"code"`
	Message    LegacyMessageCode  `json:"message"`
	SentStatus []LegacyStatusItem `json:"sent_status"`
}
type LegacyStatusOutput struct{ Body LegacyStatusBody }
