package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

var errorSetup sync.Once

// ConfigureErrors installs Huma's documented error factory once at startup.
// Framework validation and controller errors use the same documented DTO.
func ConfigureErrors() {
	errorSetup.Do(func() {
		huma.NewError = func(status int, _ string, details ...error) huma.StatusError {
			code := codeForStatus(status)
			result := &dtos.ErrorOutput{ErrorModel: huma.ErrorModel{Status: status, Title: http.StatusText(status), Detail: code.Message()}, Code: code}
			if status < http.StatusInternalServerError {
				for _, detail := range details {
					if typed, ok := detail.(huma.ErrorDetailer); ok {
						item := typed.ErrorDetail()
						result.Errors = append(result.Errors, &huma.ErrorDetail{Message: item.Message, Location: item.Location})
					}
				}
			}
			return result
		}
	})
}

func codeForStatus(status int) apperrors.Code {
	switch status {
	case http.StatusUnauthorized:
		return apperrors.CodeUnauthorized
	case http.StatusForbidden:
		return apperrors.CodeForbidden
	case http.StatusNotFound:
		return apperrors.CodeNotFound
	case http.StatusTooManyRequests:
		return apperrors.CodeRateLimited
	case http.StatusConflict:
		return apperrors.CodeConflict
	case http.StatusServiceUnavailable:
		return apperrors.CodeServiceUnavailable
	default:
		if status >= http.StatusInternalServerError {
			return apperrors.CodeInternal
		}
		return apperrors.CodeInvalidRequest
	}
}

func HTTPError(err error) huma.StatusError {
	var appError *apperrors.Error
	if errors.As(err, &appError) {
		switch appError.Code {
		case apperrors.CodeServiceUnavailable, apperrors.CodeDatabaseUnavailable:
			return huma.NewError(http.StatusServiceUnavailable, "")
		case apperrors.CodeConflict, apperrors.CodeLastAdmin:
			return &dtos.ErrorOutput{ErrorModel: huma.ErrorModel{Status: http.StatusConflict, Title: http.StatusText(http.StatusConflict), Detail: appError.Code.Message()}, Code: appError.Code}
		case apperrors.CodeCSRFInvalid:
			return &dtos.ErrorOutput{ErrorModel: huma.ErrorModel{Status: http.StatusForbidden, Title: http.StatusText(http.StatusForbidden), Detail: appError.Code.Message()}, Code: appError.Code}
		case apperrors.CodeInvalidRequest:
			return huma.NewError(http.StatusUnprocessableEntity, "")
		case apperrors.CodeUnauthorized:
			return huma.NewError(http.StatusUnauthorized, "")
		case apperrors.CodeForbidden:
			return huma.NewError(http.StatusForbidden, "")
		case apperrors.CodeNotFound:
			return huma.NewError(http.StatusNotFound, "")
		case apperrors.CodeRateLimited:
			return huma.NewError(http.StatusTooManyRequests, "")
		}
	}
	return huma.NewError(http.StatusInternalServerError, "")
}

// WriteHTTPError handles router errors using the same public error factory.
func WriteHTTPError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(huma.NewError(status, ""))
}
