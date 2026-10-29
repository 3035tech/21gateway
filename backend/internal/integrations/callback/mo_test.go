package callback

import (
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/models"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func moFixture() models.MOMessage {
	return models.MOMessage{ID: 1370, AttemptID: 166524, CorrelationID: "177627B", PhoneNumber: "5511998877665", Message: "Teste MO\nOlá! 😊", ReceivedAt: time.Date(2018, 7, 3, 20, 27, 2, 0, time.UTC)}
}
func TestLegacyMOTranslation(t *testing.T) {
	calls := 0
	client := NewWithTransport(fakeTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-21Mobile-Event-ID") != "mo:1370" || r.URL.RawQuery != "fixed=yes" {
			t.Fatal("incorrect request")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		expected := map[string]any{"Message": "Teste MO\nOlá! 😊", "Mobile21Identifier": float64(1370), "CorrelationId": "177627B", "MtId": float64(166524), "PhoneNumber": "5511998877665", "Date": "Tue Jul 03 17:27:02 BRT 2018"}
		if !reflect.DeepEqual(body, expected) {
			t.Fatalf("wrong callback body: %#v", body)
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	ok, code := client.DeliverMO(context.Background(), "https://client.example/mo?fixed=yes", moFixture())
	if !ok || code != "" || calls != 1 {
		t.Fatal(ok, code, calls)
	}
}
func TestMOUsesIndependentBasicAuth(t *testing.T) {
	message := moFixture()
	message.CredentialID = 17
	client := NewWithTransport(fakeTransport(func(r *http.Request) (*http.Response, error) {
		user, password, ok := r.BasicAuth()
		if r.Method != http.MethodPost || !ok || user != "recipient" || password != "secret" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unexpected MO request")
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	client.SetAuthResolver(func(_ context.Context, id uint64, kind models.CallbackKind) (RequestAuth, error) {
		if id != 17 || kind != models.CallbackKindMO {
			t.Fatal("wrong destination")
		}
		return RequestAuth{Mode: models.CallbackAuthBasic, Username: "recipient", Secret: "secret"}, nil
	})
	ok, code := client.DeliverMO(context.Background(), "https://client.example/mo", message)
	if !ok || code != "" {
		t.Fatal(ok, code)
	}
}
func TestMOInvalidInputsNeverReachTransport(t *testing.T) {
	calls := 0
	client := NewWithTransport(fakeTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
	for _, target := range []string{"http://client.example/mo", "https://user:secret@client.example/mo", "https://client.example:8443/mo", "https://client.example/mo#fragment"} {
		if ok, _ := client.DeliverMO(context.Background(), target, moFixture()); ok {
			t.Fatal("unsafe target allowed")
		}
	}
	for _, change := range []func(*models.MOMessage){func(m *models.MOMessage) { m.ID = 0 }, func(m *models.MOMessage) { m.AttemptID = 0 }, func(m *models.MOMessage) { m.ReceivedAt = time.Time{} }, func(m *models.MOMessage) { m.Message = string([]byte{0xff}) }, func(m *models.MOMessage) { m.Message = strings.Repeat("x", maximumMOMessageBytes+1) }, func(m *models.MOMessage) { m.PhoneNumber = "invalid" }} {
		message := moFixture()
		change(&message)
		if ok, code := client.DeliverMO(context.Background(), "https://client.example/mo", message); ok || code != models.CallbackInvalidPayload {
			t.Fatal("invalid message accepted", code)
		}
	}
	if calls != 0 {
		t.Fatal("invalid callback reached transport")
	}
}
func TestMONoAutomaticRetryOnHTTPFailure(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusTooManyRequests, http.StatusInternalServerError} {
		calls := 0
		client := NewWithTransport(fakeTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("private endpoint details"))}, nil
		}))
		if ok, code := client.DeliverMO(context.Background(), "https://client.example/mo", moFixture()); ok || code != models.CallbackHTTPError || calls != 1 {
			t.Fatal("unexpected retry or result", status, calls, code)
		}
	}
}
func TestMOLoopbackBlocked(t *testing.T) {
	client := New()
	defer client.Close()
	if ok, code := client.DeliverMO(context.Background(), "https://127.0.0.1/mo", moFixture()); ok || code != models.CallbackNetworkError {
		t.Fatal("loopback callback allowed", code)
	}
}
