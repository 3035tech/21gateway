package whitelabel

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
)

const callbackMessageID = "897db3ef-e4d9-471d-bdb5-e6d1a07936bc"
const submissionPayload = `{"send_type":"SMS","message_id":"897db3ef-e4d9-471d-bdb5-e6d1a07936bc","reference_id":"147852","submit_status_code":"ESME_ROK","submit_date":"2024-01-15T10:30:00.000Z","size":1,"callback_type":1}`
const deliveryPayload = `{"send_type":"SMS","message_id":"897db3ef-e4d9-471d-bdb5-e6d1a07936bc","reference_id":"147852","report_status":"DELIVRD","report_date":"2024-01-15T10:31:00.000Z","err":"000","esm_class":4,"callback_type":2}`
const moPayload = `{"send_type":"SMS","message_id":"897db3ef-e4d9-471d-bdb5-e6d1a07936bc","reference_id":"147852","short_message":"Olá, esta é uma resposta","source_addr":"5511999999999","report_date":"2024-01-15T10:32:00.000Z","esm_class":0,"callback_type":3}`

func TestDecodeDocumentedCallbackTypes(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		kind int
	}{
		{"submission", submissionPayload, CallbackSubmission},
		{"delivery", deliveryPayload, CallbackDelivery},
		{"mo", moPayload, CallbackMO},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := DecodeCallback([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if payload.CallbackType != tc.kind || payload.MessageID != callbackMessageID || payload.ReferenceID == nil || *payload.ReferenceID != "147852" {
				t.Fatalf("unexpected payload: %+v", payload)
			}
		})
	}
	payload, err := DecodeCallback([]byte(strings.Replace(moPayload, `"callback_type":3`, `"callback_type":3,"future_field":true`, 1)))
	if err != nil || payload.ShortMessage == nil || *payload.ShortMessage != "Olá, esta é uma resposta" {
		t.Fatal("MO content or forward-compatible field was lost", err)
	}
}

func TestNormalizeOnlyKnownDeliveryStatuses(t *testing.T) {
	for _, tc := range []struct {
		code string
		want models.DeliveryStatus
	}{
		{"DELIVRD", models.DeliveryDelivered},
		{"UNDELIV", models.DeliveryNotDelivered},
		{"EXPIRED", models.DeliveryNotDelivered},
		{"REJECTD", models.DeliveryNotDelivered},
		{"DELETED", models.DeliveryCancelled},
		{"ACCEPTD", models.DeliverySent},
		{"ENROUTE", models.DeliverySent},
	} {
		t.Run(tc.code, func(t *testing.T) {
			input, err := NormalizeDeliveryCallback([]byte(strings.Replace(deliveryPayload, "DELIVRD", tc.code, 1)))
			if err != nil || input.Body.Status != tc.want || input.Body.ProviderMessageID != callbackMessageID || !strings.HasPrefix(input.Body.EventKey, "zoug:") {
				t.Fatalf("unexpected delivery translation: %+v %v", input, err)
			}
		})
	}
	input, err := NormalizeDeliveryCallback([]byte(submissionPayload))
	if err != nil || input.Body.Status != models.DeliverySent || !input.Body.OccurredAt.Equal(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected submission translation: %+v %v", input, err)
	}
	for _, raw := range []string{strings.Replace(deliveryPayload, "DELIVRD", "UNKNOWN", 1), strings.Replace(submissionPayload, "ESME_ROK", "ESME_RSYSERR", 1), moPayload} {
		if _, err := NormalizeDeliveryCallback([]byte(raw)); !errors.Is(err, ErrUnsupportedCallback) {
			t.Fatalf("unsupported event was translated: %v", err)
		}
	}
}

func TestCallbackIdentityIsStableAcrossRetries(t *testing.T) {
	first, err := NormalizeDeliveryCallback([]byte(deliveryPayload))
	if err != nil {
		t.Fatal(err)
	}
	second, err := NormalizeDeliveryCallback([]byte(`{"callback_type":2,"esm_class":4,"err":"000","report_date":"2024-01-15T10:31:00Z","report_status":"DELIVRD","reference_id":"147852","message_id":" 897db3ef-e4d9-471d-bdb5-e6d1a07936bc","send_type":"SMS"}`))
	if err != nil || first.Body.EventKey != second.Body.EventKey || !first.Body.OccurredAt.Equal(second.Body.OccurredAt) {
		t.Fatalf("equivalent callback produced different event identity: %v", err)
	}
}

func TestDecodeRejectsMissingOrInvalidRequiredFields(t *testing.T) {
	for _, raw := range []string{
		`{`,
		`null`,
		strings.Replace(submissionPayload, `"send_type":"SMS"`, `"send_type":"MMS"`, 1),
		strings.Replace(submissionPayload, `"message_id":"`+callbackMessageID+`"`, `"message_id":"invalid"`, 1),
		strings.Replace(submissionPayload, `"reference_id":"147852",`, "", 1),
		strings.Replace(submissionPayload, `"submit_date":"2024-01-15T10:30:00.000Z"`, `"submit_date":"2024-01-15T10:30:00-03:00"`, 1),
		strings.Replace(deliveryPayload, `"esm_class":4`, `"esm_class":0`, 1),
		strings.Replace(moPayload, `"source_addr":"5511999999999",`, "", 1),
		deliveryPayload + deliveryPayload,
		strings.Repeat("x", maximumCallbackBytes+1),
	} {
		if _, err := DecodeCallback([]byte(raw)); !errors.Is(err, ErrInvalidCallback) {
			t.Fatalf("invalid callback accepted: %v", err)
		}
	}
}
