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
	MessageText   string `json:"messageText" minLength:"1" maxLength:"160" doc:"Texto do SMS. Até 160 caracteres na entrada; acentos e emojis são removidos pela normalização do gateway."`
	Destination   string `json:"destination" minLength:"11" maxLength:"11" pattern:"^[0-9]{11}$" doc:"Celular brasileiro com DDD e 9 dígitos, sem +55, espaços ou pontuação. Exemplo: 11987654321."`
	CorrelationID string `json:"correlationId,omitempty" maxLength:"50" doc:"Referência opcional do cliente, devolvida na resposta. Não é chave de idempotência."`
	ScheduledDate *int64 `json:"scheduledDate,omitempty" minimum:"0" maximum:"253402300799" doc:"Unix timestamp em segundos para envio futuro. Ausente ou zero envia imediatamente; agendamento depende de ativação no ambiente."`
}

type LegacySendInput struct {
	Body struct {
		SMS []LegacySMS `json:"sms" minItems:"1" maxItems:"200" doc:"De 1 a 200 mensagens; a resposta contém um resultado por item na mesma ordem."`
	}
}

type LegacySendResult struct {
	Status        LegacyResultStatus `json:"status" enum:"OK,NOK" doc:"OK indica aceite da tentativa; não confirma entrega ao aparelho. NOK indica falha conhecida ou resultado incerto."`
	Code          LegacyResultCode   `json:"code" enum:"5,-5,429" doc:"5: aceito; -5: falha ou resultado incerto; 429: limite de envio atingido."`
	IDSMS         string             `json:"id_sms,omitempty" doc:"Identificador interno da tentativa, para suporte e consultas posteriores."`
	Message       LegacyMessageCode  `json:"message,omitempty" doc:"Código textual de erro ou resultado, quando aplicável; SEND_OUTCOME_UNKNOWN exige reconciliação antes de qualquer reenvio."`
	CorrelationID string             `json:"correlationId,omitempty" doc:"Referência fornecida pelo cliente, quando presente na entrada."`
}

type LegacySendOutput struct{ Body []LegacySendResult }
