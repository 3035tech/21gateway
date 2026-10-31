package controllers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/providers/webhook"
	"github.com/21mobile/sms-gateway/internal/security"
)

type zougDeliveryStub struct {
	calls     int
	err       error
	provider  models.Provider
	messageID string
}

func TestZougWebhookDiagnosticLogOmitsURLTokenAndSecretHeaders(t *testing.T) {
	const token = "controlled-zoug-webhook-token-0123456789"
	const body = `{"send_type":"SMS","message_id":"897db3ef-e4d9-471d-bdb5-e6d1a07936bc","report_status":"UNKNOWN","report_date":"2026-10-28T13:12:45.000Z","err":"120","esm_class":4,"callback_type":2}`
	digest := sha256.Sum256([]byte(token))
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	router := chi.NewRouter()
	router.Use(security.ClientIPResolver{}.Middleware)
	NewProviderCallbackController(&zougDeliveryStub{}, &zougMOStub{}, true, whitelabel.NewWebhookAdapter(hex.EncodeToString(digest[:]), []string{"18.232.239.74"})).Register(router)
	request := httptest.NewRequest(http.MethodPost, "/webhooks/zoug/"+token, strings.NewReader(body))
	request.RemoteAddr = "18.232.239.74:1234"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "secret-authorization")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !strings.Contains(output.String(), "UNKNOWN") || !strings.Contains(output.String(), `"http_status":204`) || strings.Contains(output.String(), token) || strings.Contains(output.String(), "secret-authorization") {
		t.Fatalf("diagnostic log missing request/response or exposed a secret: %s", output.String())
	}
}

func (s *zougDeliveryStub) AcceptProviderDelivery(_ context.Context, provider models.Provider, input dtos.DeliveryInput) error {
	s.calls++
	s.provider = provider
	s.messageID = input.Body.ProviderMessageID
	return s.err
}

type alternateWebhookAdapter struct{}

func (alternateWebhookAdapter) Code() models.Provider { return models.Provider("test-supplier") }
func (alternateWebhookAdapter) AuthenticateToken(token string) bool {
	return token == "" || token == "alternate-secret"
}
func (alternateWebhookAdapter) AllowPeer(peer netip.Addr) bool { return peer.IsValid() }
func (alternateWebhookAdapter) AuthenticateBody(r *http.Request, _ []byte) bool {
	return r.Header.Get("X-Test-Signature") == "valid"
}
func (alternateWebhookAdapter) Decode(raw []byte) (webhook.Event, error) {
	var payload struct {
		ExternalID string `json:"external_id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.ExternalID == "" {
		return webhook.Event{}, errors.New("invalid")
	}
	var input dtos.DeliveryInput
	input.Body.ProviderMessageID = payload.ExternalID
	input.Body.EventKey = "test-supplier:event-1"
	input.Body.Status = models.DeliveryDelivered
	input.Body.OccurredAt = time.Now().UTC()
	return webhook.Event{Delivery: &input}, nil
}

func TestWebhookSelectsAuthenticatedAdapterBeforeDecoding(t *testing.T) {
	delivery := &zougDeliveryStub{}
	router := chi.NewRouter()
	router.Use(security.ClientIPResolver{}.Middleware)
	NewProviderCallbackController(delivery, &zougMOStub{}, false, whitelabel.NewWebhookAdapter(strings.Repeat("a", 64), []string{"192.0.2.1"}), alternateWebhookAdapter{}).Register(router)
	request := httptest.NewRequest(http.MethodPost, "/webhooks/test-supplier/alternate-secret", strings.NewReader(`{"external_id":"opaque-42"}`))
	request.RemoteAddr = "192.0.2.1:1234"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Test-Signature", "valid")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || delivery.provider != models.Provider("test-supplier") || delivery.messageID != "opaque-42" {
		t.Fatalf("status=%d provider=%s id=%s", response.Code, delivery.provider, delivery.messageID)
	}
	request = httptest.NewRequest(http.MethodPost, "/webhooks/test-supplier/alternate-secret", strings.NewReader(`{"external_id":"opaque-43"}`))
	request.RemoteAddr = "192.0.2.1:1234"
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || delivery.calls != 1 {
		t.Fatal("unsigned body reached provider decoder")
	}
	request = httptest.NewRequest(http.MethodPost, "/webhooks/test-supplier", strings.NewReader(`{"external_id":"opaque-44"}`))
	request.RemoteAddr = "192.0.2.1:1234"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Test-Signature", "valid")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || delivery.messageID != "opaque-44" {
		t.Fatal("header-only webhook authentication failed")
	}
	request = httptest.NewRequest(http.MethodPost, "/webhooks/zoug/alternate-secret", strings.NewReader(`{"external_id":"opaque-42"}`))
	request.RemoteAddr = "192.0.2.1:1234"
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || delivery.calls != 2 {
		t.Fatal("webhook selected adapter from JSON or crossed provider route")
	}
}

type zougMOStub struct {
	calls int
	err   error
}

func (s *zougMOStub) AcceptProviderMO(_ context.Context, _ models.Provider, _ webhook.MOMessage) error {
	s.calls++
	return s.err
}

func TestZougWebhookAuthenticationAndPersistence(t *testing.T) {
	const token = "controlled-zoug-webhook-token-0123456789"
	digest := sha256.Sum256([]byte(token))
	const id = "897db3ef-e4d9-471d-bdb5-e6d1a07936bc"
	delivery := `{"send_type":"SMS","message_id":"` + id + `","report_status":"DELIVRD","report_date":"2026-10-28T13:12:45.000Z","err":"120","esm_class":4,"callback_type":2}`
	mo := `{"send_type":"SMS","message_id":"` + id + `","short_message":"Obrigado","report_date":"2026-10-28T13:14:08.000Z","esm_class":0,"callback_type":3}`
	for _, tc := range []struct {
		name, token, peer, body string
		deliveryErr             error
		want, deliveries, mos   int
	}{
		{"wrong token", "wrong", "18.232.239.74:1234", delivery, nil, 404, 0, 0},
		{"wrong IP", token, "198.51.100.1:1234", delivery, nil, 403, 0, 0},
		{"DLR persisted", token, "18.232.239.74:1234", delivery, nil, 204, 1, 0},
		{"MO persisted", token, "18.232.239.74:1234", mo, nil, 204, 0, 1},
		{"database failed", token, "18.232.239.74:1234", delivery, errors.New("private database detail"), 503, 1, 0},
		{"invalid body", token, "18.232.239.74:1234", `{}`, nil, 422, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &zougDeliveryStub{err: tc.deliveryErr}
			m := &zougMOStub{}
			router := chi.NewRouter()
			router.Use(security.ClientIPResolver{}.Middleware)
			NewProviderCallbackController(d, m, false, whitelabel.NewWebhookAdapter(hex.EncodeToString(digest[:]), []string{"18.232.239.74"})).Register(router)
			request := httptest.NewRequest(http.MethodPost, "/webhooks/zoug/"+tc.token, strings.NewReader(tc.body))
			request.RemoteAddr = tc.peer
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.want || d.calls != tc.deliveries || m.calls != tc.mos || strings.Contains(response.Body.String(), "private database detail") {
				t.Fatalf("status=%d delivery=%d mo=%d body=%q", response.Code, d.calls, m.calls, response.Body.String())
			}
		})
	}
}
