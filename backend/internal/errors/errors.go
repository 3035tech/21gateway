// Package errors defines application error codes independently of HTTP.
package errors

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

type Code string

const (
	CodeNone                      Code = ""
	CodeInvalidRequest            Code = "INVALID_REQUEST"
	CodeUnauthorized              Code = "UNAUTHORIZED"
	CodeForbidden                 Code = "FORBIDDEN"
	CodeNotFound                  Code = "NOT_FOUND"
	CodeRateLimited               Code = "RATE_LIMITED"
	CodeServiceUnavailable        Code = "SERVICE_UNAVAILABLE"
	CodeInternal                  Code = "INTERNAL_ERROR"
	CodeConfigurationInvalid      Code = "CONFIGURATION_INVALID"
	CodeDatabaseUnavailable       Code = "DATABASE_UNAVAILABLE"
	CodeInvalidDestination        Code = "INVALID_DESTINATION"
	CodeInvalidMessage            Code = "INVALID_MESSAGE"
	CodeBatchUnsupported          Code = "BATCH_UNSUPPORTED"
	CodeScheduleUnsupported       Code = "SCHEDULE_UNSUPPORTED"
	CodeProviderUnauthorized      Code = "PROVIDER_UNAUTHORIZED"
	CodeInsufficientBalance       Code = "INSUFFICIENT_BALANCE"
	CodeProviderRejected          Code = "PROVIDER_REJECTED"
	CodeUnknownOutcome            Code = "UNKNOWN_OUTCOME"
	CodeConflict                  Code = "CONFLICT"
	CodeLastAdmin                 Code = "LAST_ADMIN_REQUIRED"
	CodeCSRFInvalid               Code = "CSRF_INVALID"
	CodeTestDestinationNotAllowed Code = "TEST_DESTINATION_NOT_ALLOWED"
	CodeTestNotConfigured         Code = "TEST_NOT_CONFIGURED"
	CodeInvalidCurrentPassword    Code = "INVALID_CURRENT_PASSWORD"
	CodeMFARequired               Code = "MFA_REQUIRED"
	CodeMFAInvalid                Code = "MFA_INVALID"
	CodeInvalidResetToken         Code = "INVALID_RESET_TOKEN"
	CodeInvitationDeliveryFailed  Code = "INVITATION_DELIVERY_FAILED"
)

//go:embed locales/*.json
var catalogs embed.FS

var messages = loadMessages()

func loadMessages() map[string]map[Code]string {
	result := make(map[string]map[Code]string)
	for _, language := range []string{"pt-BR", "en-US"} {
		contents, err := catalogs.ReadFile("locales/" + language + ".json")
		if err != nil {
			panic(err)
		}
		var catalog map[Code]string
		if err := json.Unmarshal(contents, &catalog); err != nil {
			panic(err)
		}
		result[language] = catalog
	}
	return result
}

func (c Code) Message() string { return c.MessageFor("pt-BR") }

func (c Code) MessageFor(language string) string {
	locale := "pt-BR"
	if strings.HasPrefix(strings.ToLower(language), "en") {
		locale = "en-US"
	}
	if value := messages[locale][c]; value != "" {
		return value
	}
	return messages[locale][CodeInternal]
}

type Error struct {
	Code  Code
	cause error
}

func Wrap(code Code, cause error) *Error { return &Error{Code: code, cause: cause} }
func (e *Error) Error() string           { return fmt.Sprintf("%s: %s", e.Code, e.Code.Message()) }
func (e *Error) Unwrap() error           { return e.cause }
