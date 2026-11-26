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

const (
	MaximumGSM7Characters   = 160
	MaximumLatin1Characters = 70
	MaximumUCS2Characters   = 70
)

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
	if isLatin1(message) {
		return models.SMSEncodingLatin1
	}
	return models.SMSEncodingUCS2
}

func SelectBatchEncoding(messages []string) models.SMSEncoding {
	allGSM7, allLatin1 := true, true
	for _, message := range messages {
		if _, err := gsm7.Encode([]byte(message)); err != nil {
			allGSM7 = false
		}
		if !isLatin1(message) {
			allLatin1 = false
		}
	}
	if allGSM7 {
		return models.SMSEncodingGSM7
	}
	if allLatin1 {
		return models.SMSEncodingLatin1
	}
	return models.SMSEncodingUCS2
}

func FitsProviderMessageLength(message string, encoding models.SMSEncoding) bool {
	maximum := 0
	switch encoding {
	case models.SMSEncodingGSM7:
		encoded, err := gsm7.Encode([]byte(message))
		return err == nil && len(encoded) <= MaximumGSM7Characters
	case models.SMSEncodingLatin1:
		maximum = MaximumLatin1Characters
	case models.SMSEncodingUCS2:
		maximum = MaximumUCS2Characters
	default:
		return false
	}
	return utf8.RuneCountInString(message) <= maximum
}

func isLatin1(message string) bool {
	for _, character := range message {
		if character > 0xFF {
			return false
		}
	}
	return true
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
