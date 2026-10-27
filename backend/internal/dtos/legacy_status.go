package dtos

import "github.com/21mobile/sms-gateway/internal/models"

const (
	LegacyStatusPageSize                   = 100
	LegacyCodeNoRecords  LegacyResultCode  = 6
	LegacyQueryOK        LegacyMessageCode = "REQUIRED_OK"
	// The manual does not name the empty-result message; this is provisional.
	LegacyQueryEmpty LegacyMessageCode = "NO_RECORDS"
)

type SMSDeliveryStatusItem struct {
	CorrelationID string                `json:"correlationId" doc:"Referência informada no envio. Somente tentativas com correlationId entram nesta consulta."`
	Destination   string                `json:"destination" doc:"Destino registrado pelo gateway, com prefixo 55 nos números válidos."`
	SentDate      string                `json:"sentDate" doc:"Data do envio no fuso America/Sao_Paulo, formato AAAA-MM-DD HH:MM:SS-03."`
	UnixSentDate  string                `json:"unixSentDate" doc:"Instante do envio em Unix timestamp, como string decimal."`
	Status        models.DeliveryStatus `json:"status" doc:"Estado de entrega recebido por DLR. SENT não equivale a DELIVERED."`
}
type LegacyStatusItem = SMSDeliveryStatusItem

type SMSDeliveryStatusResponse struct {
	Status     LegacyResultStatus      `json:"status" doc:"OK para consulta concluída, inclusive quando a lista está vazia."`
	Code       LegacyResultCode        `json:"code" doc:"5 quando há estados; 6 quando não há novos registros."`
	Message    LegacyMessageCode       `json:"message" doc:"REQUIRED_OK quando há estados; NO_RECORDS para lista vazia (texto provisório)."`
	SentStatus []SMSDeliveryStatusItem `json:"sent_status" doc:"Até 100 estados de entrega ainda não consultados; a leitura consome os registros retornados."`
}
type LegacyStatusBody = SMSDeliveryStatusResponse

type SMSDeliveryStatusOutput struct{ Body SMSDeliveryStatusResponse }
type LegacyStatusOutput = SMSDeliveryStatusOutput
