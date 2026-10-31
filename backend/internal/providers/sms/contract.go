package sms

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/warthog618/sms/encoding/gsm7"
)

const MaximumBatchMessages = 200 // Legacy /v1/send contract.
const MaximumProviderMessageIDBytes = 255

type CallbackLevel string

const (
	CallbackNone CallbackLevel = "NONE"
	CallbackAll  CallbackLevel = "ALL"
)

type SendRequest struct {
	Type          models.Channel
	Message       string
	Encoding      models.SMSEncoding
	Destination   string
	ReferenceID   string
	Index         string
	CallbackLevel CallbackLevel
}
type SendResponse struct {
	MessageID string
	Status    string
}
type BatchMessage struct {
	Message     string
	Destination string
	ReferenceID string
	Index       string
}
type BatchRequest struct {
	Type     models.Channel
	Encoding models.SMSEncoding
	Messages []BatchMessage
}
type BatchResult struct {
	MessageID string
	ErrorCode apperrors.Code
}

type Sender interface {
	Enabled() bool
	Send(context.Context, SendRequest) (SendResponse, error)
}
type BatchSender interface {
	SendBatch(context.Context, BatchRequest) ([]BatchResult, error)
}

type Adapter interface {
	Code() models.Provider
	PrepareCredential(context.Context, models.APICredential) (Sender, error)
	PrepareCompany(context.Context, models.Company) (Sender, error)
}

func SelectEncoding(message string) models.SMSEncoding {
	if _, err := gsm7.Encode([]byte(message)); err == nil {
		return models.SMSEncodingGSM7
	}
	return models.SMSEncodingUCS2
}

func ValidMessageID(id string) bool {
	if id == "" || len(id) > MaximumProviderMessageIDBytes || !utf8.ValidString(id) || strings.TrimSpace(id) != id {
		return false
	}
	for _, character := range id {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}
