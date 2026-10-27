package dtos

import "time"

// LegacyMOCallback preserves the exact field spelling in the supplied manual.
// CorrelationId is a string: the manual's unquoted alphanumeric example is invalid JSON.
type LegacyMOCallback struct {
	Message            string `json:"Message"`
	Mobile21Identifier uint64 `json:"Mobile21Identifier"`
	CorrelationID      string `json:"CorrelationId"`
	MTID               uint64 `json:"MtId"`
	PhoneNumber        string `json:"PhoneNumber"`
	Date               string `json:"Date"`
}

type MOInput struct {
	Body struct {
		EventKey          string    `json:"eventKey" minLength:"1" maxLength:"200"`
		CredentialID      uint64    `json:"credentialId" minimum:"1"`
		ProviderMessageID string    `json:"providerMessageId" format:"uuid"`
		PhoneNumber       string    `json:"phoneNumber" pattern:"^[0-9]{1,15}$"`
		Message           string    `json:"message" minLength:"1" maxLength:"4096"`
		ReceivedAt        time.Time `json:"receivedAt"`
	}
}

type LegacyMOQueryInput struct {
	Date int64 `query:"date" required:"true" minimum:"0" maximum:"253402300799" doc:"Unix timestamp em segundos do dia a consultar"`
}
type LegacyMOItem struct {
	FromAddress      string `json:"fromAdress"`
	ReceivedDate     string `json:"receivedDate"`
	UnixReceivedDate string `json:"unixReceivedDate"`
	Message          string `json:"msg"`
	IDMO             string `json:"idMo"`
	IDMT             string `json:"idMt"`
}
type LegacyMOBody struct {
	Status  LegacyResultStatus `json:"status"`
	Code    LegacyResultCode   `json:"code"`
	Message LegacyMessageCode  `json:"message"`
	ListMO  []LegacyMOItem     `json:"list_mo"`
}
type LegacyMOOutput struct{ Body LegacyMOBody }
