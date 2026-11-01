package pontal

import (
	"errors"
	"regexp"
	"strings"

	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

var ErrInvalidPDU = errors.New("invalid Pontal SMPP PDU")
var ErrSubmitRejected = errors.New("Pontal SMPP submit rejected")
var ErrInvalidSubmit = errors.New("invalid Pontal SMPP submit")

var receiptStatusPattern = regexp.MustCompile(`(?:^|\s)stat:([A-Z]+)(?:\s|$)`)
var receiptIDPattern = regexp.MustCompile(`(?:^|\s)id:\S+`)
var numericAddressPattern = regexp.MustCompile(`^[0-9]+$`)

// SubmitInput describes only the fields observed in the Node submit_sm call.
// Source address, TON/NPI and multipart behavior need supplier confirmation.
type SubmitInput struct {
	Destination    string
	Message        string
	Encoding       models.SMSEncoding
	RequestReceipt bool
}

// BuildSubmit creates one short-message PDU. It rejects content that needs
// multipart handling; that mapping is not yet confirmed for Pontal.
func BuildSubmit(input SubmitInput) (*pdu.SubmitSM, error) {
	if input.Destination == "" || len(input.Destination) > data.SM_ADDR_LEN || !numericAddressPattern.MatchString(input.Destination) || strings.TrimSpace(input.Message) == "" {
		return nil, ErrInvalidSubmit
	}
	var encoding data.Encoding
	switch input.Encoding {
	case models.SMSEncodingGSM7:
		encoding = data.GSM7BIT
	case models.SMSEncodingUCS2:
		encoding = data.UCS2
	default:
		return nil, ErrInvalidSubmit
	}
	message, err := pdu.NewShortMessageWithEncoding(input.Message, encoding)
	if err != nil {
		return nil, ErrInvalidSubmit
	}
	encoded, err := message.GetMessageData()
	if err != nil || len(encoded) > data.SM_GSM_MSG_LEN {
		return nil, ErrInvalidSubmit
	}
	request := pdu.NewSubmitSM().(*pdu.SubmitSM)
	if err := request.DestAddr.SetAddress(input.Destination); err != nil {
		return nil, ErrInvalidSubmit
	}
	request.Message = message
	if input.RequestReceipt {
		request.RegisteredDelivery = data.SM_SMSC_RECEIPT_REQUESTED
	}
	return request, nil
}

type InboundKind string

const (
	InboundReceipt InboundKind = "receipt"
	InboundReply   InboundKind = "reply"
)

// Inbound is a transport-level observation. The caller must persist and
// reconcile it before acknowledging deliver_sm. RawStatus is not a gateway
// delivery status; its mapping requires Pontal's contract.
type Inbound struct {
	Kind              InboundKind
	ProviderMessageID string
	RawStatus         string
	MessageState      *byte
	SourceAddress     string
	Message           string
}

// DecodeSubmitResponse extracts the SMSC message ID only after a successful
// submit_sm_resp. A transport failure or missing response is not a rejection.
func DecodeSubmitResponse(value pdu.PDU) (string, error) {
	response, ok := value.(*pdu.SubmitSMResp)
	if !ok || response == nil {
		return "", ErrInvalidPDU
	}
	if !response.IsOk() {
		return "", ErrSubmitRejected
	}
	if !smsprovider.ValidMessageID(response.MessageID) {
		return "", ErrInvalidPDU
	}
	return response.MessageID, nil
}

// DecodeDeliver distinguishes SMSC receipts from mobile-originated messages.
// It never maps a missing or unknown receipt status to a final failure.
func DecodeDeliver(value pdu.PDU) (Inbound, error) {
	delivery, ok := value.(*pdu.DeliverSM)
	if !ok || delivery == nil {
		return Inbound{}, ErrInvalidPDU
	}
	text, err := delivery.Message.GetMessage()
	if err != nil {
		return Inbound{}, ErrInvalidPDU
	}
	receiptID, hasReceiptID := delivery.OptionalParameters[pdu.TagReceiptedMessageID]
	_, hasMessageState := delivery.OptionalParameters[pdu.TagMessageStateOption]
	isReceipt := delivery.EsmClass&data.SM_SMSC_DLV_RCPT_TYPE != 0 || hasReceiptID || hasMessageState || (receiptIDPattern.MatchString(text) && receiptStatusPattern.MatchString(text))
	if isReceipt {
		if !hasReceiptID {
			return Inbound{}, ErrInvalidPDU
		}
		id := strings.TrimRight(receiptID.String(), "\x00")
		if !smsprovider.ValidMessageID(id) {
			return Inbound{}, ErrInvalidPDU
		}
		result := Inbound{Kind: InboundReceipt, ProviderMessageID: id}
		if status := receiptStatusPattern.FindStringSubmatch(text); len(status) == 2 {
			result.RawStatus = status[1]
		}
		if state, ok := delivery.OptionalParameters[pdu.TagMessageStateOption]; ok {
			if len(state.Data) != 1 {
				return Inbound{}, ErrInvalidPDU
			}
			value := state.Data[0]
			result.MessageState = &value
		}
		return result, nil
	}
	address := delivery.SourceAddr.Address()
	if address == "" || strings.TrimSpace(text) == "" {
		return Inbound{}, ErrInvalidPDU
	}
	return Inbound{Kind: InboundReply, SourceAddress: address, Message: text}, nil
}
