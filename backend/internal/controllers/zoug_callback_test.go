package controllers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/security"
)

type zougDeliveryStub struct {
	calls int
	err   error
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
	NewZougCallbackController(&zougDeliveryStub{}, &zougMOStub{}, hex.EncodeToString(digest[:]), []string{"18.232.239.74"}, true).Register(router)
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

func (s *zougDeliveryStub) AcceptWhiteLabelDelivery(_ context.Context, _ []byte) error {
	s.calls++
	return s.err
}

type zougMOStub struct {
	calls int
	err   error
}

func (s *zougMOStub) AcceptWhiteLabelMO(_ context.Context, _ whitelabel.CallbackPayload) error {
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
			NewZougCallbackController(d, m, hex.EncodeToString(digest[:]), []string{"18.232.239.74"}).Register(router)
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
