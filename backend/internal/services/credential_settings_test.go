package services

import (
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"strings"
	"testing"
)

func TestCredentialSettingsKeepLegacyTokenPrivate(t *testing.T) {
	service := NewCredentialService(nil)
	enabled := true
	ips := []string{"203.0.113.10"}
	credential := models.APICredential{Username: "client", ProviderTokenEncrypted: "encrypted-legacy-token", CompanyProviderTokenLegacyFallback: true}
	body := dtos.CredentialBody{WhitelistEnabled: &enabled, AllowedIPs: &ips}
	if err := service.applySettings(&credential, body); err != nil {
		t.Fatal(err)
	}
	if err := service.applySettings(&credential, dtos.CredentialBody{}); err != nil || credential.ProviderTokenEncrypted != "encrypted-legacy-token" {
		t.Fatal("omitted settings did not preserve token")
	}
	dto, _ := json.Marshal(dtos.CredentialDTO(credential))
	if strings.Contains(string(dto), credential.ProviderTokenEncrypted) {
		t.Fatal("provider token leaked from DTO")
	}
	if err := service.applySettings(&models.APICredential{}, dtos.CredentialBody{WhitelistEnabled: &enabled}); err == nil {
		t.Fatal("empty enabled whitelist accepted")
	}
}
