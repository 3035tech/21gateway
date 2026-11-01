package pontal

import (
	"errors"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

func TestDecodeSubmitResponseRequiresSuccessfulAcknowledgement(t *testing.T) {
	response := pdu.NewSubmitSMResp().(*pdu.SubmitSMResp)
	response.MessageID = "pontal-123"
	if id, err := DecodeSubmitResponse(response); err != nil || id != response.MessageID {
		t.Fatalf("id=%q err=%v", id, err)
	}
	response.CommandStatus = data.ESME_RSYSERR
	if _, err := DecodeSubmitResponse(response); !errors.Is(err, ErrSubmitRejected) {
		t.Fatalf("expected rejection, got %v", err)
	}
	response.CommandStatus = data.ESME_ROK
	response.MessageID = ""
	if _, err := DecodeSubmitResponse(response); !errors.Is(err, ErrInvalidPDU) {
		t.Fatalf("expected missing ID, got %v", err)
	}
}

func TestBuildSubmitUsesObservedNodeFields(t *testing.T) {
	request, err := BuildSubmit(SubmitInput{Destination: "5511999999999", Message: "Teste 21", Encoding: models.SMSEncodingGSM7, RequestReceipt: true})
	if err != nil {
		t.Fatal(err)
	}
	message, err := request.Message.GetMessage()
	if err != nil || message != "Teste 21" || request.DestAddr.Address() != "5511999999999" || request.SourceAddr.Address() != "" || request.RegisteredDelivery != data.SM_SMSC_RECEIPT_REQUESTED {
		t.Fatalf("unexpected submit PDU: destination=%q source=%q receipt=%d message=%q err=%v", request.DestAddr.Address(), request.SourceAddr.Address(), request.RegisteredDelivery, message, err)
	}
	ucs2, err := BuildSubmit(SubmitInput{Destination: "5511999999999", Message: "漢", Encoding: models.SMSEncodingUCS2})
	if err != nil || ucs2.Message.Encoding().DataCoding() != data.UCS2Coding || ucs2.RegisteredDelivery != data.SM_SMSC_RECEIPT_NOT_REQUESTED {
		t.Fatalf("unexpected UCS2 submit: %+v err=%v", ucs2, err)
	}
	for _, input := range []SubmitInput{
		{Destination: "not-a-number", Message: "Teste", Encoding: models.SMSEncodingGSM7},
		{Destination: "5511999999999", Message: "", Encoding: models.SMSEncodingGSM7},
		{Destination: "5511999999999", Message: "Teste", Encoding: "unknown"},
		{Destination: "5511999999999", Message: strings.Repeat("a", data.SM_GSM_MSG_LEN+1), Encoding: models.SMSEncodingGSM7},
	} {
		if _, err := BuildSubmit(input); !errors.Is(err, ErrInvalidSubmit) {
			t.Fatalf("expected invalid submit for %+v, got %v", input, err)
		}
	}
}

func TestDecodeDeliverKeepsReceiptStatusRaw(t *testing.T) {
	delivery := pdu.NewDeliverSM().(*pdu.DeliverSM)
	delivery.EsmClass = data.SM_SMSC_DLV_RCPT_TYPE
	delivery.RegisterOptionalParam(pdu.Field{Tag: pdu.TagReceiptedMessageID, Data: []byte("pontal-123\x00")})
	delivery.RegisterOptionalParam(pdu.Field{Tag: pdu.TagMessageStateOption, Data: []byte{2}})
	message, err := pdu.NewShortMessage("id:pontal-123 stat:DELIVRD err:000")
	if err != nil {
		t.Fatal(err)
	}
	delivery.Message = message
	decoded, err := DecodeDeliver(delivery)
	if err != nil || decoded.Kind != InboundReceipt || decoded.ProviderMessageID != "pontal-123" || decoded.RawStatus != "DELIVRD" || decoded.MessageState == nil || *decoded.MessageState != 2 || decoded.Message != "" {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
	delivery.OptionalParameters = nil
	if _, err := DecodeDeliver(delivery); !errors.Is(err, ErrInvalidPDU) {
		t.Fatalf("receipt without ID must not become MO: %v", err)
	}
	delivery.EsmClass = 0
	if _, err := DecodeDeliver(delivery); !errors.Is(err, ErrInvalidPDU) {
		t.Fatalf("textual receipt without ID must not become MO: %v", err)
	}
}

func TestDecodeDeliverSeparatesMobileOriginatedMessage(t *testing.T) {
	delivery := pdu.NewDeliverSM().(*pdu.DeliverSM)
	if err := delivery.SourceAddr.SetAddress("5511999999999"); err != nil {
		t.Fatal(err)
	}
	message, err := pdu.NewShortMessage("resposta")
	if err != nil {
		t.Fatal(err)
	}
	delivery.Message = message
	decoded, err := DecodeDeliver(delivery)
	if err != nil || decoded.Kind != InboundReply || decoded.SourceAddress != "5511999999999" || decoded.Message != "resposta" || decoded.ProviderMessageID != "" {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
}
