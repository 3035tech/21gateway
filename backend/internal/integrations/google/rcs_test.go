package google

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/providers/rcs"
	"golang.org/x/oauth2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func testRequest() rcs.TextRequest {
	return rcs.TextRequest{AgentID: "agent-1", Phone: "+5511999999999", MessageID: "35b1cabf-e20b-4a34-a7fe-b5e6962b83f5", Text: "Olá"}
}

func testClient(transport http.RoundTripper) *Client {
	return NewWithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token", Expiry: time.Now().Add(time.Hour)}), transport)
}

func TestSendTextGoogleWireContract(t *testing.T) {
	input := testRequest()
	calls := 0
	client := testClient(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Method != http.MethodPost || request.URL.Host != "rcsbusinessmessaging.googleapis.com" || request.URL.Path != "/v1/phones/"+input.Phone+"/agentMessages" {
			t.Fatalf("unexpected URL: %s", request.URL)
		}
		if request.URL.Query().Get("agentId") != input.AgentID || request.URL.Query().Get("messageId") != input.MessageID {
			t.Fatal("missing agent or message ID")
		}
		if request.Header.Get("Authorization") != "Bearer test-token" || request.Header.Get("Content-Type") != "application/json" {
			t.Fatal("missing OAuth or content type")
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != `{"contentMessage":{"text":"Olá"}}` {
			t.Fatalf("unexpected body: %s", body)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"name":"phones/+5511999999999/agentMessages/35b1cabf-e20b-4a34-a7fe-b5e6962b83f5"}`)), Header: make(http.Header)}, nil
	}))
	result, err := client.SendText(context.Background(), input)
	if err != nil || result.MessageID != input.MessageID || calls != 1 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
	}
}

func TestSendTextRejectsInvalidInputBeforeNetwork(t *testing.T) {
	calls := 0
	client := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected") }))
	for _, change := range []func(*rcs.TextRequest){
		func(r *rcs.TextRequest) { r.Phone = "5511999999999" },
		func(r *rcs.TextRequest) { r.MessageID = "not-a-uuid" },
		func(r *rcs.TextRequest) { r.Text = strings.Repeat("a", maximumTextCharacters+1) },
	} {
		input := testRequest()
		change(&input)
		if _, err := client.SendText(context.Background(), input); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	}
	if calls != 0 {
		t.Fatal("invalid input reached network")
	}
}

func TestSendTextDoesNotRetryUnknownOutcome(t *testing.T) {
	calls := 0
	client := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("network failure containing secret")
	}))
	_, err := client.SendText(context.Background(), testRequest())
	if !errors.Is(err, ErrUnknownOutcome) || calls != 1 || strings.Contains(err.Error(), "secret") {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestSendTextRejectsWrongAcknowledgement(t *testing.T) {
	client := testClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"name":"phones/+5511888888888/agentMessages/35b1cabf-e20b-4a34-a7fe-b5e6962b83f5"}`)), Header: make(http.Header)}, nil
	}))
	_, err := client.SendText(context.Background(), testRequest())
	if !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("expected unknown outcome, got %v", err)
	}
}
