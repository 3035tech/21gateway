package controllers

import (
	"net/http"
	"testing"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

func TestProviderErrorsPreserveTheirCode(t *testing.T) {
	for _, code := range []apperrors.Code{apperrors.CodeProviderUnauthorized, apperrors.CodeInsufficientBalance, apperrors.CodeProviderRejected, apperrors.CodeUnknownOutcome} {
		result := HTTPError(apperrors.Wrap(code, nil))
		body, ok := result.(*dtos.ErrorOutput)
		if !ok || result.GetStatus() != http.StatusBadGateway || body.Code != code {
			t.Fatalf("provider code %s collapsed to generic error: %#v", code, result)
		}
	}
}
