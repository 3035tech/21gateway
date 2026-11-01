package services

import (
	"testing"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
)

func TestPlannedRoutesCannotReplaceSMSOrDuplicateAChannel(t *testing.T) {
	for _, routes := range [][]dtos.PlannedProviderRoute{
		{{Channel: models.ChannelSMS, Provider: models.ProviderZenvia}},
		{{Channel: models.ChannelWhatsApp, Provider: models.ProviderGoogle}},
		{{Channel: models.ChannelRCS, Provider: models.ProviderGoogle}, {Channel: models.ChannelRCS, Provider: models.ProviderZenvia}},
	} {
		_, err := validatePlannedRoutes(&routes)
		assertCode(t, err, apperrors.CodeInvalidRequest)
	}
	valid := []dtos.PlannedProviderRoute{{Channel: models.ChannelRCS, Provider: models.ProviderGoogle}, {Channel: models.ChannelWhatsApp, Provider: models.ProviderZenvia}}
	parsed, err := validatePlannedRoutes(&valid)
	if err != nil || len(parsed) != len(valid) {
		t.Fatalf("valid planned routes rejected: %v", err)
	}
}
