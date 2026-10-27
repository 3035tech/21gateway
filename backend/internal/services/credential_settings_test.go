package services

import (
	"encoding/base64"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"strings"
	"testing"
)

func TestCredentialSettingsKeepTokenPrivateAndExplicit(t *testing.T) {
	vault, _ := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	service := NewCredentialServiceWithVault(nil, vault)
	token := "private-token"
	enabled := true
	ips := []string{"203.0.113.10"}
	credential := models.APICredential{Username: "client"}
	body := dtos.CredentialBody{ProviderToken: &token, WhitelistEnabled: &enabled, AllowedIPs: &ips}
	if err := service.applySettings(&credential, body); err != nil {
		t.Fatal(err)
	}
	encrypted := credential.ProviderTokenEncrypted
	if encrypted == token || encrypted == "" {
		t.Fatal("plaintext provider token stored")
	}
	if err := service.applySettings(&credential, dtos.CredentialBody{}); err != nil || credential.ProviderTokenEncrypted != encrypted {
		t.Fatal("omitted settings did not preserve token")
	}
	dto, _ := json.Marshal(dtos.CredentialDTO(credential))
	if strings.Contains(string(dto), token) || strings.Contains(string(dto), encrypted) {
		t.Fatal("provider token leaked from DTO")
	}
	if err := service.applySettings(&credential, dtos.CredentialBody{ClearProviderToken: true}); err != nil || credential.ProviderTokenEncrypted != "" {
		t.Fatal("explicit token removal failed")
	}
	if err := service.applySettings(&models.APICredential{}, dtos.CredentialBody{WhitelistEnabled: &enabled}); err == nil {
		t.Fatal("empty enabled whitelist accepted")
	}
}
