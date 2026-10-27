package dtos

type LegacyResultStatus string
type LegacyResultCode int
type LegacyMessageCode string

const (
	LegacyStatusOK             LegacyResultStatus = "OK"
	LegacyStatusNOK            LegacyResultStatus = "NOK"
	LegacyCodeAccepted         LegacyResultCode   = 5
	LegacyCodeFailed           LegacyResultCode   = -5
	LegacyCodeRateLimited      LegacyResultCode   = 429
	LegacyInvalidNumber        LegacyMessageCode  = "INVALID_NUMBER_FORMAT"
	LegacyInvalidMessage       LegacyMessageCode  = "INVALID_MESSAGE_FORMAT"
	LegacyInvalidRequest       LegacyMessageCode  = "INVALID_REQUEST"
	LegacyBatchUnsupported     LegacyMessageCode  = "BATCH_NOT_SUPPORTED"
	LegacyScheduleUnsupported  LegacyMessageCode  = "SCHEDULING_NOT_SUPPORTED"
	LegacyInsufficientBalance  LegacyMessageCode  = "INSUFFICIENT_BALANCE"
	LegacyProviderUnauthorized LegacyMessageCode  = "PROVIDER_UNAUTHORIZED"
	LegacyProviderRejected     LegacyMessageCode  = "PROVIDER_REJECTED"
	LegacyUnknownOutcome       LegacyMessageCode  = "SEND_OUTCOME_UNKNOWN"
	LegacyRateLimited          LegacyMessageCode  = "TPS_LIMIT_EXCEEDED"
	LegacyGeneralError         LegacyMessageCode  = "GENERAL_ERROR"
)

type LegacySMS struct {
	MessageText   string `json:"messageText" minLength:"1" maxLength:"160"`
	Destination   string `json:"destination" pattern:"^[0-9]{11}$"`
	CorrelationID string `json:"correlationId,omitempty" maxLength:"50"`
	ScheduledDate *int64 `json:"scheduledDate,omitempty"`
}

type LegacySendInput struct {
	Body struct {
		SMS []LegacySMS `json:"sms" minItems:"1" maxItems:"200"`
	}
}

type LegacySendResult struct {
	Status        LegacyResultStatus `json:"status" enum:"OK,NOK"`
	Code          LegacyResultCode   `json:"code" enum:"5,-5,429"`
	IDSMS         string             `json:"id_sms,omitempty"`
	Message       LegacyMessageCode  `json:"message,omitempty"`
	CorrelationID string             `json:"correlationId,omitempty"`
}

type LegacySendOutput struct{ Body []LegacySendResult }
