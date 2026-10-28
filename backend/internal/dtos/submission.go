package dtos

import (
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
)

type SMSSubmissionQuery struct {
	IDSMS uint64 `query:"id_sms" minimum:"1" doc:"id_sms devolvido por POST /v1/send. Somente envios da própria credencial podem ser consultados."`
}

type SMSSubmissionStatus struct {
	IDSMS          string                 `json:"id_sms" doc:"Identificador do envio devolvido no aceite."`
	State          models.SubmissionState `json:"state" enum:"pending,accepted,rejected,unknown" doc:"pending: na fila; accepted: aceito pelo gateway; rejected: falha conhecida; unknown: resultado incerto, sem reenvio automático."`
	FailureCode    string                 `json:"failureCode,omitempty" doc:"Código estável da falha, quando existir."`
	DeliveryStatus string                 `json:"deliveryStatus,omitempty" doc:"Último estado de entrega recebido por DLR, quando existir. Ausência não confirma entrega."`
	CreatedAt      time.Time              `json:"createdAt" doc:"Instante de aceite pelo gateway 21 Mobile, em UTC."`
	DispatchedAt   *time.Time             `json:"dispatchedAt,omitempty" doc:"Instante em que a chamada ao fornecedor começou, em UTC, quando aplicável."`
}

type SMSSubmissionOutput struct{ Body SMSSubmissionStatus }
