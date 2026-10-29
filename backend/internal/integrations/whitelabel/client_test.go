package whitelabel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

const testBaseURL = "https://provider.example.com"

func TestHTTPTranslationAndAcknowledgement(t *testing.T) {
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", func(r *http.Request) (*http.Response, error) {
		if r.Header.Get(tokenHeader) != "test-provider-token" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("incorrect request metadata")
		}
		var body SendRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Type != MessageTypeSMS || body.CallbackLevel != CallbackNone || body.Message != "Olá!" {
			t.Errorf("incorrect translation: %+v", body)
		}
		return httpmock.NewJsonResponse(http.StatusOK, SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: StatusSent})
	})
	client, err := NewWithTransport(testBaseURL, "test-provider-token", transport)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	result, err := client.Send(context.Background(), SendRequest{Type: MessageTypeSMS, Message: "Olá!", Encoding: SelectEncoding("Olá!"), Destination: "5511987654321", Index: "1", CallbackLevel: CallbackNone})
	if err != nil || result.Status != StatusSent || transport.GetTotalCallCount() != 1 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, transport.GetTotalCallCount())
	}
}

func TestProviderErrorsAreSafeAndNotRetried(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		code   apperrors.Code
	}{
		{"balance", http.StatusPaymentRequired, `{"error":"private details"}`, apperrors.CodeInsufficientBalance},
		{"provider credential", http.StatusUnauthorized, `{"error":"private token"}`, apperrors.CodeProviderUnauthorized},
		{"tps", http.StatusTooManyRequests, `{}`, apperrors.CodeRateLimited},
		{"internal error is uncertain", http.StatusInternalServerError, `{}`, apperrors.CodeUnknownOutcome},
		{"invalid acknowledgement", http.StatusOK, `{"message_id":"invalid","status":"sent"}`, apperrors.CodeUnknownOutcome},
		{"invalid response", http.StatusOK, `not-json`, apperrors.CodeUnknownOutcome},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := httpmock.NewMockTransport()
			transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", httpmock.NewStringResponder(tc.status, tc.body))
			client, _ := NewWithTransport(testBaseURL, "test-token", transport)
			defer client.Close()
			_, err := client.Send(context.Background(), SendRequest{Type: MessageTypeSMS, Message: "test"})
			var typed *apperrors.Error
			if !errors.As(err, &typed) || typed.Code != tc.code || transport.GetTotalCallCount() != 1 {
				t.Fatalf("err=%v calls=%d", err, transport.GetTotalCallCount())
			}
		})
	}
}

func TestUnexpectedAcknowledgementKeepsValidProviderIDForReconciliation(t *testing.T) {
	transport := httpmock.NewMockTransport()
	const id = "550e8400-e29b-41d4-a716-446655440000"
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", httpmock.NewStringResponder(http.StatusOK, `{"message_id":"`+id+`","status":"queued"}`))
	client, _ := NewWithTransport(testBaseURL, "test-token", transport)
	defer client.Close()
	response, err := client.Send(context.Background(), SendRequest{Type: MessageTypeSMS, Message: "test"})
	var typed *apperrors.Error
	if !errors.As(err, &typed) || typed.Code != apperrors.CodeUnknownOutcome || response.MessageID != id {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}

func TestUnexpectedAcknowledgementResponseBodyDiagnostic(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	const body = `{"message_id":"550e8400-e29b-41d4-a716-446655440000","status":"queued","details":{"reason":"provider pending"}}`
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", httpmock.NewStringResponder(http.StatusOK, body))
	client, err := NewWithTransport(testBaseURL, "private-token", transport)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	input := SendRequest{Message: "private-message", Destination: "5511999999999", Index: "42"}
	_, _ = client.Send(context.Background(), input)
	if strings.Contains(logs.String(), `"response_body"`) {
		t.Fatal("diagnostic enabled by default")
	}
	logs.Reset()
	client.SetLogUnexpectedResponseBody(true)
	_, _ = client.ForToken("private-token").Send(context.Background(), input)
	var event map[string]any
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		if event["msg"] == "Zoug SMS response unavailable" {
			break
		}
	}
	if event["response_body"] != body || event["http_status"] != float64(http.StatusOK) {
		t.Fatalf("response diagnostic missing: %s", logs.String())
	}
	for _, secret := range []string{"private-token", "private-message", "5511999999999"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("request data logged: %s", secret)
		}
	}
}

func TestRedirectDoesNotLeakProviderToken(t *testing.T) {
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", func(r *http.Request) (*http.Response, error) {
		response := httpmock.NewStringResponse(http.StatusTemporaryRedirect, "")
		response.Header.Set("Location", "https://untrusted.example.com/capture")
		return response, nil
	})
	client, _ := NewWithTransport(testBaseURL, "secret-test-token", transport)
	defer client.Close()
	_, err := client.Send(context.Background(), SendRequest{})
	if err == nil || transport.GetTotalCallCount() != 1 {
		t.Fatal("provider redirect followed")
	}
}

func TestNetworkFailureIsUncertainAndNotRetried(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
	client, _ := NewWithTransport(testBaseURL, "private-token", transport)
	defer client.Close()
	_, err := client.Send(context.Background(), SendRequest{Index: "42", Message: "private-message", Destination: "5511999999999"})
	var typed *apperrors.Error
	if !errors.As(err, &typed) || typed.Code != apperrors.CodeUnknownOutcome || transport.GetTotalCallCount() != 1 {
		t.Fatalf("err=%v calls=%d", err, transport.GetTotalCallCount())
	}
	output := logs.String()
	if !strings.Contains(output, `"provider_index":"42"`) || !strings.Contains(output, `"reason":"transport"`) || !strings.Contains(output, `"error":`) {
		t.Fatalf("missing diagnostic fields: %s", output)
	}
	for _, secret := range []string{"private-token", "private-message", "5511999999999"} {
		if strings.Contains(output, secret) {
			t.Fatalf("sensitive data logged: %s", secret)
		}
	}
}

func TestEncodingPreservesUnicodeAndGSMExtensionCharacters(t *testing.T) {
	for _, tc := range []struct {
		text string
		want Encoding
	}{{"Ola 123", EncodingGSM7}, {"Olá ç 😀", EncodingUCS2}, {"Valor € ^ {}", EncodingGSM7}} {
		if got := SelectEncoding(tc.text); got != tc.want {
			t.Fatalf("%q: got %s want %s", tc.text, got, tc.want)
		}
	}
}
