package google

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"testing"
)

func TestParseWebhookAuthenticatesDecodedPayload(t *testing.T) {
	data := []byte(`{"eventType":"DELIVERED","messageId":"id"}`)
	mac := hmac.New(sha512.New, []byte("client-token"))
	_, _ = mac.Write(data)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	request := []byte(`{"message":{"data":"` + base64.StdEncoding.EncodeToString(data) + `"}}`)
	callback, err := ParseWebhook(WebhookMessageCallback, signature, "client-token", request)
	if err != nil || callback.Type != WebhookMessageCallback || string(callback.Data) != string(data) {
		t.Fatalf("callback=%+v err=%v", callback, err)
	}
	if _, err := ParseWebhook(WebhookMessageCallback, signature, "wrong-token", request); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("expected invalid signature, got %v", err)
	}
	if _, err := ParseWebhook(WebhookMessageCallback, signature, "client-token", []byte(`{"message":{"data":"broken"}}`)); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("expected invalid payload, got %v", err)
	}
}

func TestVerificationSecretChecksToken(t *testing.T) {
	secret, err := VerificationSecret(WebhookVerification, "token", []byte(`{"clientToken":"token","secret":"challenge"}`))
	if err != nil || secret != "challenge" {
		t.Fatalf("secret=%q err=%v", secret, err)
	}
	if _, err := VerificationSecret(WebhookVerification, "different", []byte(`{"clientToken":"token","secret":"challenge"}`)); !errors.Is(err, ErrInvalidWebhook) {
		t.Fatalf("expected invalid challenge, got %v", err)
	}
}
