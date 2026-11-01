package pontal

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

// SubmitDecision is the only safe conclusion from a submit_sm response.
// Submit() returning nil merely queues a PDU inside gosmpp. If the bind closes,
// a write times out, or the response is missing, acceptance is unknown.
type SubmitDecision struct {
	State             models.SubmissionState
	ProviderMessageID string
	CommandStatus     data.CommandStatusType
}

func ClassifySubmitResponse(value pdu.PDU) SubmitDecision {
	response, ok := value.(*pdu.SubmitSMResp)
	if !ok || response == nil {
		return SubmitDecision{State: models.SubmissionUnknown}
	}
	if !response.IsOk() {
		return SubmitDecision{State: models.SubmissionRejected, CommandStatus: response.CommandStatus}
	}
	id, err := DecodeSubmitResponse(response)
	if err != nil {
		return SubmitDecision{State: models.SubmissionUnknown}
	}
	return SubmitDecision{State: models.SubmissionAccepted, ProviderMessageID: id}
}
