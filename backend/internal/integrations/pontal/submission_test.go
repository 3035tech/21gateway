package pontal

import (
	"testing"

	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

func TestClassifySubmitResponseDoesNotRetryUnknownDelivery(t *testing.T) {
	for _, value := range []pdu.PDU{nil, pdu.NewDeliverSM(), pdu.NewSubmitSMResp()} {
		decision := ClassifySubmitResponse(value)
		if decision.State != models.SubmissionUnknown || decision.ProviderMessageID != "" {
			t.Fatalf("missing or malformed response must stay unknown: %+v", decision)
		}
	}

	rejected := pdu.NewSubmitSMResp().(*pdu.SubmitSMResp)
	rejected.CommandStatus = data.ESME_RSYSERR
	rejected.MessageID = "ignored-on-rejection"
	decision := ClassifySubmitResponse(rejected)
	if decision.State != models.SubmissionRejected || decision.CommandStatus != data.ESME_RSYSERR || decision.ProviderMessageID != "" {
		t.Fatalf("explicit rejection was lost: %+v", decision)
	}

	accepted := pdu.NewSubmitSMResp().(*pdu.SubmitSMResp)
	accepted.MessageID = "pontal-123"
	decision = ClassifySubmitResponse(accepted)
	if decision.State != models.SubmissionAccepted || decision.ProviderMessageID != accepted.MessageID {
		t.Fatalf("confirmed acceptance was lost: %+v", decision)
	}
}
