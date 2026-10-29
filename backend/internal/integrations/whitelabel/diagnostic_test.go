package whitelabel

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestDiagnosticsNeverUseDeliverableDestinationAndCaptureResponses(t *testing.T) {
	transport := httpmock.NewMockTransport()
	responder := func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if json.Unmarshal(body, &payload) == nil {
			if destination, ok := payload["destination_addr"].(string); ok && destination != "invalid" {
				t.Fatalf("deliverable destination in diagnostic: %q", destination)
			}
		}
		return httpmock.NewStringResponse(http.StatusUnprocessableEntity, `{"status":"INVALID_REQUEST","token":"private-token"}`), nil
	}
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", responder)
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send/batch", responder)
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send/unknown", responder)
	transport.RegisterResponder(http.MethodGet, testBaseURL+"/api/send", responder)
	client, _ := NewWithTransport(testBaseURL, "private-token", transport)
	results := client.Diagnose(context.Background(), "run", 1, 2, "")
	if len(results) != 16 || transport.GetTotalCallCount() != 16 {
		t.Fatalf("results=%d calls=%d", len(results), transport.GetTotalCallCount())
	}
	for _, result := range results {
		if result.HTTPStatus != http.StatusUnprocessableEntity || strings.Contains(result.ResponseBody, "private-token") || !strings.Contains(result.ResponseBody, "INVALID_REQUEST") {
			t.Fatalf("bad diagnostic result: %+v", result)
		}
	}
}

func TestDiagnosticAllowlistedDestinationCasesStillHaveInvalidBody(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	transport := httpmock.NewMockTransport()
	responder := func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "5511999999999") {
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil || payload["type"] != "INVALID" || payload["encoding"] != "INVALID" {
				t.Fatalf("unsafe valid-destination probe: %s", body)
			}
		}
		response := httpmock.NewStringResponse(http.StatusUnprocessableEntity, `{"destination_addr":"5511999999999","status":"INVALID_REQUEST"}`)
		response.Header.Set("X-API-TOKEN", "private-token")
		response.Header.Set("Location", "https://example.test/5511999999999?key=private-token")
		return response, nil
	}
	for _, path := range []string{"/api/send", "/api/send/batch", "/api/send/unknown"} {
		transport.RegisterResponder(http.MethodPost, testBaseURL+path, responder)
	}
	transport.RegisterResponder(http.MethodGet, testBaseURL+"/api/send", responder)
	client, _ := NewWithTransport(testBaseURL, "private-token", transport)
	results := client.Diagnose(context.Background(), "run", 1, 2, "11999999999")
	if len(results) != 18 || transport.GetTotalCallCount() != 18 {
		t.Fatalf("results=%d calls=%d", len(results), transport.GetTotalCallCount())
	}
	for _, result := range results {
		if strings.Contains(result.ResponseBody, "5511999999999") {
			t.Fatal("destination leaked in diagnostic response")
		}
	}
	for _, secret := range []string{"5511999999999", "11999999999", "private-token"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("diagnostic log leaked %s", secret)
		}
	}
}

func TestDiagnosticStopsOnUnexpectedAcceptance(t *testing.T) {
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", httpmock.NewStringResponder(http.StatusOK, `{"status":"SUCCESS"}`))
	client, _ := NewWithTransport(testBaseURL, "private-token", transport)
	results := client.Diagnose(context.Background(), "run", 1, 2, "11999999999")
	if len(results) != 1 || transport.GetTotalCallCount() != 1 {
		t.Fatalf("diagnostic continued after acceptance: results=%d calls=%d", len(results), transport.GetTotalCallCount())
	}
}
