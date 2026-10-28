package whitelabel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
)

const maximumCallbackBytes = 64 << 10

const (
	CallbackSubmission = 1
	CallbackDelivery   = 2
	CallbackMO         = 3
)

var ErrInvalidCallback = errors.New("invalid white label callback")
var ErrUnsupportedCallback = errors.New("unsupported white label callback")

// CallbackPayload is the wire format documented by Zoug/Devyx on 11/02/2026.
// It deliberately accepts additional JSON fields for forward compatibility.
type CallbackPayload struct {
	SendType         string  `json:"send_type"`
	MessageID        string  `json:"message_id"`
	ReferenceID      *string `json:"reference_id"`
	CallbackType     int     `json:"callback_type"`
	SubmitStatusCode string  `json:"submit_status_code"`
	SubmitDate       string  `json:"submit_date"`
	Size             *int    `json:"size"`
	ReportStatus     string  `json:"report_status"`
	ReportDate       string  `json:"report_date"`
	ErrorCode        *string `json:"err"`
	ESMClass         *int    `json:"esm_class"`
	ShortMessage     *string `json:"short_message"`
	SourceAddress    *string `json:"source_addr"`
}

func DecodeCallback(raw []byte) (CallbackPayload, error) {
	var payload CallbackPayload
	if len(raw) == 0 || len(raw) > maximumCallbackBytes {
		return payload, ErrInvalidCallback
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&payload); err != nil {
		return CallbackPayload{}, ErrInvalidCallback
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return CallbackPayload{}, ErrInvalidCallback
	}
	id, err := uuid.Parse(strings.TrimSpace(payload.MessageID))
	if err != nil || id == uuid.Nil || payload.SendType != string(MessageTypeSMS) {
		return CallbackPayload{}, ErrInvalidCallback
	}
	payload.MessageID = id.String()
	switch payload.CallbackType {
	case CallbackSubmission:
		if payload.SubmitStatusCode == "" || payload.Size == nil || *payload.Size < 1 || parseCallbackTime(payload.SubmitDate).IsZero() {
			return CallbackPayload{}, ErrInvalidCallback
		}
	case CallbackDelivery:
		if payload.ReportStatus == "" || payload.ErrorCode == nil || payload.ESMClass == nil || *payload.ESMClass != 4 || parseCallbackTime(payload.ReportDate).IsZero() {
			return CallbackPayload{}, ErrInvalidCallback
		}
	case CallbackMO:
		if payload.ShortMessage == nil || *payload.ShortMessage == "" || payload.ESMClass == nil || parseCallbackTime(payload.ReportDate).IsZero() {
			return CallbackPayload{}, ErrInvalidCallback
		}
	default:
		return CallbackPayload{}, ErrUnsupportedCallback
	}
	return payload, nil
}

func parseCallbackTime(value string) time.Time {
	date, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || value == "" || value[len(value)-1] != 'Z' {
		return time.Time{}
	}
	return date.UTC()
}

// NormalizeDeliveryCallback maps only documented, unambiguous DLR outcomes.
// MO requires a confirmed account and outbound-message association before use.
func NormalizeDeliveryCallback(raw []byte) (dtos.DeliveryInput, error) {
	var input dtos.DeliveryInput
	payload, err := DecodeCallback(raw)
	if err != nil {
		return input, err
	}
	var status models.DeliveryStatus
	var occurredAt time.Time
	switch payload.CallbackType {
	case CallbackSubmission:
		switch payload.SubmitStatusCode {
		case "ESME_ROK":
			status = models.DeliverySent
		case "UNKNOWN_ERROR":
			status = models.DeliveryNotDelivered
		default:
			return input, ErrUnsupportedCallback
		}
		occurredAt = parseCallbackTime(payload.SubmitDate)
	case CallbackDelivery:
		occurredAt = parseCallbackTime(payload.ReportDate)
		switch payload.ReportStatus {
		case "DELIVRD":
			status = models.DeliveryDelivered
		case "UNDELIV", "EXPIRED", "REJECTD":
			status = models.DeliveryNotDelivered
		case "DELETED":
			status = models.DeliveryCancelled
		case "ACCEPTD", "ENROUTE":
			status = models.DeliverySent
		default:
			return input, ErrUnsupportedCallback
		}
	default:
		return input, ErrUnsupportedCallback
	}
	occurredAt = occurredAt.Truncate(time.Microsecond)
	identity := struct {
		Type      int
		MessageID string
		Status    string
		Date      string
		ErrorCode string
		Size      int
	}{Type: payload.CallbackType, MessageID: payload.MessageID, Status: payload.ReportStatus, Date: occurredAt.Format(time.RFC3339Nano)}
	if payload.CallbackType == CallbackSubmission {
		identity.Status = payload.SubmitStatusCode
		identity.Size = *payload.Size
	} else {
		identity.ErrorCode = *payload.ErrorCode
	}
	canonical, _ := json.Marshal(identity)
	digest := sha256.Sum256(canonical)
	input.Body.EventKey = "zoug:" + hex.EncodeToString(digest[:])
	input.Body.ProviderMessageID = payload.MessageID
	input.Body.Status = status
	input.Body.OccurredAt = occurredAt
	return input, nil
}
