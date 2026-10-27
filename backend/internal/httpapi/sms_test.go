package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"gorm.io/gorm"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
)

type testCredentials struct{ value models.APICredential }

func (r testCredentials) FindByUsername(ctx context.Context, username string) (models.APICredential, error) {
	if username == r.value.Username {
		return r.value, nil
	}
	return models.APICredential{}, gorm.ErrRecordNotFound
}

type testAttempts struct{ created, completed []models.SMSAttempt }

func (r *testAttempts) Create(ctx context.Context, attempt *models.SMSAttempt) error {
	attempt.ID = 42
	r.created = append(r.created, *attempt)
	return nil
}
func (r *testAttempts) Complete(ctx context.Context, attempt models.SMSAttempt) error {
	r.completed = append(r.completed, attempt)
	return nil
}

func TestLegacySendHTTPPipeline(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, username, password, body string
		want                           int
		calls                          int
	}{
		{"accepted", "client", "test-password-only", `{"sms":[{"messageText":"Ola!","destination":"11987654321","correlationId":"ref-42"}]}`, http.StatusOK, 1},
		{"no auth", "", "", `{"sms":[]}`, http.StatusUnauthorized, 0},
		{"wrong password", "client", "wrong-password", `{"sms":[]}`, http.StatusUnauthorized, 0},
		{"unknown user", "unknown", "test-password-only", `{"sms":[]}`, http.StatusUnauthorized, 0},
		{"invalid destination", "client", "test-password-only", `{"sms":[{"messageText":"Ola!","destination":"invalid"}]}`, http.StatusOK, 0},
		{"scheduled", "client", "test-password-only", `{"sms":[{"messageText":"Ola!","destination":"11987654321","scheduledDate":1900000000}]}`, http.StatusOK, 0},
		{"bad json", "client", "test-password-only", `{"sms":`, http.StatusUnprocessableEntity, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := httpmock.NewMockTransport()
			transport.RegisterResponder(http.MethodPost, "https://provider.example.com/api/send", func(r *http.Request) (*http.Response, error) {
				return httpmock.NewJsonResponse(http.StatusOK, whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440000", Status: whitelabel.StatusSent})
			})
			provider, err := whitelabel.NewWithTransport("https://provider.example.com", "test-token", transport)
			if err != nil {
				t.Fatal(err)
			}
			defer provider.Close()
			repo := &testAttempts{}
			auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true}})
			if err != nil {
				t.Fatal(err)
			}
			router := New(nil, true, func(context.Context) error { return nil }, Dependencies{SMS: services.NewSMSService(repo, provider), Auth: auth})
			request := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.username != "" {
				request.SetBasicAuth(tc.username, tc.password)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.want || transport.GetTotalCallCount() != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, transport.GetTotalCallCount(), response.Body.String())
			}
			if tc.calls == 1 {
				var results []dtos.LegacySendResult
				if err := json.Unmarshal(response.Body.Bytes(), &results); err != nil {
					t.Fatal(err)
				}
				if results[0].IDSMS != "42" || results[0].CorrelationID != "ref-42" || results[0].Code != dtos.LegacyCodeAccepted || repo.completed[0].CredentialID != 7 {
					t.Fatalf("wrong result=%+v persisted=%+v", results, repo.completed)
				}
			}
			if strings.Contains(response.Body.String(), "test-token") || strings.Contains(response.Body.String(), hash) {
				t.Fatal("secret in response")
			}
		})
	}
}

type testBatchAttempts struct{ testAttempts }

func (r *testBatchAttempts) CreateBatch(_ context.Context, attempts []*models.SMSAttempt) error {
	for i, v := range attempts {
		v.ID = uint64(42 + i)
		r.created = append(r.created, *v)
	}
	return nil
}
func (r *testBatchAttempts) CompleteBatch(_ context.Context, attempts []models.SMSAttempt) error {
	r.completed = append(r.completed, attempts...)
	return nil
}
func TestLegacyBatchAccountTokenAndWhitelistHTTP(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	vault, err := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := vault.Encrypt("client", "private-account-token")
	if err != nil {
		t.Fatal(err)
	}
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder("POST", "https://provider.example/api/send/batch", func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-API-TOKEN") != "private-account-token" {
			t.Fatal("wrong account token")
		}
		return httpmock.NewStringResponder(200, `{"success":[{"position":0,"message_id":"550e8400-e29b-41d4-a716-446655440001"}],"errors":[],"total_items":1}`)(r)
	})
	client, _ := whitelabel.NewWithTransport("https://provider.example", "unused-shared-token", transport)
	defer client.Close()
	repo := &testBatchAttempts{}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true, ProviderTokenEncrypted: encrypted, WhitelistEnabled: true, AllowedIPs: []string{"192.0.2.0/24"}}})
	if err != nil {
		t.Fatal(err)
	}
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{SMS: services.NewSMSService(repo, services.NewAccountSMSProvider(client, vault)), Auth: auth})
	body := `{"sms":[{"messageText":"Ola!","destination":"11987654321","correlationId":"first"},{"messageText":"Ola!","destination":"","correlationId":"second"}]}`
	call := func(peer string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("POST", "/v1/send", strings.NewReader(body))
		request.RemoteAddr = peer
		request.SetBasicAuth("client", "test-password-only")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Forwarded-For", "192.0.2.10")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	response := call("192.0.2.10:1234")
	var outcomes []dtos.LegacySendResult
	if err := json.Unmarshal(response.Body.Bytes(), &outcomes); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(outcomes) != 2 || outcomes[0].IDSMS != "42" || outcomes[0].CorrelationID != "first" || outcomes[1].CorrelationID != "second" || outcomes[1].Message != dtos.LegacyInvalidNumber || len(repo.completed) != 1 {
		t.Fatal("legacy batch HTTP translation failed")
	}
	if strings.Contains(response.Body.String(), "private-account-token") {
		t.Fatal("provider token leaked")
	}
	blocked := call("203.0.113.1:1234")
	if blocked.Code != http.StatusForbidden || transport.GetTotalCallCount() != 1 {
		t.Fatal("forwarded header bypassed the IP whitelist")
	}
}
