package callback

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/models"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) Do(r *http.Request) (*http.Response, error) { return f(r) }
func TestLegacyCallbackTranslation(t *testing.T) {
	client := NewWithTransport(fakeTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" || r.URL.Query().Get("idSms") != "42" || r.URL.Query().Get("correlationId") != "ref&?" || r.URL.Query().Get("status") != "DELIVERED" || r.URL.Query().Get("fixed") != "yes" || r.Header.Get("X-21Mobile-Event-ID") != "9" {
			t.Fatal("invalid translated callback")
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	ok, code := client.Deliver(context.Background(), models.CallbackJob{URL: "https://client.example/dlr?fixed=yes", EventID: 9, Status: models.DeliveryDelivered}, models.SMSAttempt{ID: 42, CorrelationID: "ref&?"})
	if !ok || code != "" {
		t.Fatal(ok, code)
	}
}
func TestCallbackRejectsRedirectAndUnsafeURL(t *testing.T) {
	calls := 0
	client := NewWithTransport(fakeTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	for _, url := range []string{"http://client.example/dlr", "https://user:secret@client.example/dlr", "https://client.example:8443/dlr"} {
		ok, _ := client.Deliver(context.Background(), models.CallbackJob{URL: url}, models.SMSAttempt{})
		if ok {
			t.Fatal("unsafe URL accepted")
		}
	}
	if calls != 0 {
		t.Fatal("unsafe request reached transport")
	}
	ok, code := client.Deliver(context.Background(), models.CallbackJob{URL: "https://client.example/dlr"}, models.SMSAttempt{})
	if ok || code != models.CallbackHTTPError {
		t.Fatal("redirect must not count as delivered")
	}
}
func TestCallbackSSRFBlocksLoopback(t *testing.T) {
	client := New()
	defer client.Close()
	ok, code := client.Deliver(context.Background(), models.CallbackJob{URL: "https://127.0.0.1/dlr"}, models.SMSAttempt{})
	if ok || code != models.CallbackNetworkError {
		t.Fatal("loopback request allowed")
	}
}

func TestDLRUsesConfiguredHeaderWithoutChangingLegacyQuery(t *testing.T) {
	client := NewWithTransport(fakeTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.Header.Get("X-Client-Token") != "secret" || r.Header.Get("X-21Mobile-Event-ID") != "9" || r.URL.Query().Get("status") != "DELIVERED" {
			t.Fatal("unexpected DLR request")
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	client.SetAuthResolver(func(_ context.Context, id uint64, kind models.CallbackKind) (RequestAuth, error) {
		if id != 7 || kind != models.CallbackKindDLR {
			t.Fatal("wrong destination")
		}
		return RequestAuth{Mode: models.CallbackAuthHeader, HeaderName: "X-Client-Token", Secret: "secret"}, nil
	})
	ok, code := client.Deliver(context.Background(), models.CallbackJob{CredentialID: 7, URL: "https://client.example/dlr", EventID: 9, Status: models.DeliveryDelivered}, models.SMSAttempt{ID: 42})
	if !ok || code != "" {
		t.Fatal(ok, code)
	}
}
