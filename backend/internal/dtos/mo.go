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

type MORepliesQueryInput struct {
	Date int64 `query:"date" required:"true" minimum:"0" maximum:"253402300799" doc:"Unix timestamp em segundos de qualquer instante do dia desejado em America/Sao_Paulo; a consulta cobre o dia inteiro."`
}
type LegacyMOQueryInput = MORepliesQueryInput

type MOReplyItem struct {
	FromAddress      string `json:"fromAdress" doc:"Telefone que enviou a resposta. Use a grafia fromAdress exatamente como publicada na API."`
	ReceivedDate     string `json:"receivedDate" doc:"Data de recebimento no fuso America/Sao_Paulo, formato AAAA-MM-DD HH:MM:SS-03."`
	UnixReceivedDate string `json:"unixReceivedDate" doc:"Instante de recebimento em Unix timestamp, como string decimal."`
	Message          string `json:"msg" doc:"Texto da resposta recebida, sem normalização."`
	IDMO             string `json:"idMo" doc:"Identificador interno da resposta, como string decimal."`
	IDMT             string `json:"idMt" doc:"Identificador interno do envio original associado, como string decimal."`
}
type LegacyMOItem = MOReplyItem

type MORepliesResponse struct {
	Status  LegacyResultStatus `json:"status" doc:"OK para consulta concluída, inclusive quando a lista está vazia."`
	Code    LegacyResultCode   `json:"code" doc:"5 quando há respostas; 6 quando não há registros no dia."`
	Message LegacyMessageCode  `json:"message" doc:"REQUIRED_OK quando há respostas; NO_RECORDS para lista vazia (texto provisório)."`
	ListMO  []MOReplyItem      `json:"list_mo" doc:"Respostas recebidas no dia; a consulta não consome os registros."`
}
type LegacyMOBody = MORepliesResponse

type MORepliesOutput struct{ Body MORepliesResponse }
type LegacyMOOutput = MORepliesOutput
