package dtos

import (
	"github.com/danielgtaylor/huma/v2"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

// ErrorOutput extends RFC 9457 Problem Details with a stable application code.
type ErrorOutput struct {
	huma.ErrorModel
	Code apperrors.Code `json:"code" enum:"INVALID_REQUEST,UNAUTHORIZED,FORBIDDEN,NOT_FOUND,RATE_LIMITED,SERVICE_UNAVAILABLE,INTERNAL_ERROR,CONFLICT,LAST_ADMIN_REQUIRED,CSRF_INVALID"`
}
