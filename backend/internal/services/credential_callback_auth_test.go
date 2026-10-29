package services

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

func TestCallbackAuthSecretIsEncryptedAndNeverReturned(t *testing.T) {
	vault, err := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	service := NewCredentialServiceWithVault(nil, vault)
	secret := "private-secret"
	auth, err := service.applyCallbackAuth(models.CallbackAuth{Mode: models.CallbackAuthNone}, &dtos.CallbackAuthBody{Mode: models.CallbackAuthHeader, HeaderName: "X-Client-Token", Secret: &secret}, "gw_client", "dlr")
	if err != nil || auth.SecretEncrypted == "" || strings.Contains(auth.SecretEncrypted, secret) {
		t.Fatal("secret not encrypted", err)
	}
	plain, err := vault.Decrypt("callback-auth:dlr:gw_client", auth.SecretEncrypted)
	if err != nil || plain != secret {
		t.Fatal("secret not recoverable for delivery", err)
	}
	if _, err := vault.Decrypt("callback-auth:mo:gw_client", auth.SecretEncrypted); err == nil {
		t.Fatal("DLR secret decrypted in MO domain")
	}
	view, err := json.Marshal(dtos.CredentialDTO(models.APICredential{CallbackAuth: auth}))
	if err != nil || strings.Contains(string(view), secret) || strings.Contains(string(view), auth.SecretEncrypted) {
		t.Fatal("secret leaked in DTO", err)
	}
	kept, err := service.applyCallbackAuth(auth, &dtos.CallbackAuthBody{Mode: models.CallbackAuthHeader, HeaderName: "X-Client-Token"}, "gw_client", "dlr")
	if err != nil || kept.SecretEncrypted != auth.SecretEncrypted {
		t.Fatal("empty secret did not preserve existing value", err)
	}
}

func TestCallbackAuthRejectsUnsafeConfiguration(t *testing.T) {
	service := NewCredentialService(nil)
	secret := "secret"
	for _, body := range []dtos.CallbackAuthBody{
		{Mode: models.CallbackAuthHeader, HeaderName: "Content-Type", Secret: &secret},
		{Mode: models.CallbackAuthHeader, HeaderName: "Bad\r\nName", Secret: &secret},
		{Mode: models.CallbackAuthBasic, Username: "user:name", Secret: &secret},
		{Mode: models.CallbackAuthBasic, Username: "user"},
	} {
		if _, err := service.applyCallbackAuth(models.CallbackAuth{}, &body, "gw_client", "dlr"); err == nil {
			t.Fatalf("accepted unsafe config: %s", body.Mode)
		}
	}
}
