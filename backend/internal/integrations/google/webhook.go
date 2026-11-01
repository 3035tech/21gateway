package google

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
)

const (
	WebhookVerification    = "verification"
	WebhookMessageCallback = "message_callback"
	WebhookAgentCallback   = "agent_callback"
	WebhookTypeHeader      = "X-Goog-Webhook-Type"
	WebhookSignatureHeader = "X-Goog-Signature"
	maximumWebhookBytes    = 256 << 10
)

var ErrInvalidWebhook = errors.New("invalid RCS webhook")

type Callback struct {
	Type string
	Data json.RawMessage // Authenticated Google payload; persist before HTTP 200 when a route is activated.
}

// ParseWebhook validates a bounded, complete Google webhook body. The caller owns
// durable storage, event deduplication and acknowledgement; no HTTP route is mounted yet.
func ParseWebhook(kind, signature, clientToken string, body []byte) (Callback, error) {
	if clientToken == "" || len(body) == 0 || len(body) > maximumWebhookBytes {
		return Callback{}, ErrInvalidWebhook
	}
	if kind != WebhookMessageCallback && kind != WebhookAgentCallback {
		return Callback{}, ErrInvalidWebhook
	}
	var envelope struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Message.Data == "" {
		return Callback{}, ErrInvalidWebhook
	}
	data, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil || len(data) == 0 || len(data) > maximumWebhookBytes || !json.Valid(data) {
		return Callback{}, ErrInvalidWebhook
	}
	provided, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || len(provided) != sha512.Size {
		return Callback{}, ErrInvalidWebhook
	}
	mac := hmac.New(sha512.New, []byte(clientToken))
	_, _ = mac.Write(data)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return Callback{}, ErrInvalidWebhook
	}
	return Callback{Type: kind, Data: data}, nil
}

// VerificationSecret validates the Google challenge without echoing untrusted data.
func VerificationSecret(kind, clientToken string, body []byte) (string, error) {
	if kind != WebhookVerification || clientToken == "" || len(body) == 0 || len(body) > maximumWebhookBytes {
		return "", ErrInvalidWebhook
	}
	var challenge struct {
		ClientToken string `json:"clientToken"`
		Secret      string `json:"secret"`
	}
	if json.Unmarshal(body, &challenge) != nil || challenge.Secret == "" || !hmac.Equal([]byte(challenge.ClientToken), []byte(clientToken)) {
		return "", ErrInvalidWebhook
	}
	return challenge.Secret, nil
}
