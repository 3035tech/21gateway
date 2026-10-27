package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
)

func TestCredentialCheckUsesLegacyAuthAndPeerIPWithoutSending(t *testing.T) {
	hash, err := security.HashPassword("authorized-test-password")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{
		ID: 7, Username: "client", PasswordHash: hash, Active: true,
		WhitelistEnabled: true, AllowedIPs: []string{"192.0.2.10/32"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{Auth: auth})
	for _, tc := range []struct {
		name, remote, password string
		want                   int
	}{
		{"authorized", "192.0.2.10:4321", "authorized-test-password", http.StatusNoContent},
		{"wrong password", "192.0.2.10:4321", "wrong", http.StatusUnauthorized},
		{"wrong peer IP", "198.51.100.5:4321", "authorized-test-password", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/auth/check", nil)
			request.RemoteAddr = tc.remote
			request.SetBasicAuth("client", tc.password)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestCredentialCheckUsesTrustedCloudflareClientIP(t *testing.T) {
	hash, err := security.HashPassword("authorized-test-password")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{
		ID: 7, Username: "client", PasswordHash: hash, Active: true,
		WhitelistEnabled: true, AllowedIPs: []string{"192.0.2.10/32"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := security.NewClientIPResolver([]string{"10.42.0.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{Auth: auth, ClientIPResolver: resolver})
	for _, tc := range []struct {
		name, remote, header string
		want                 int
	}{
		{"trusted peer and allowed client", "10.42.0.3:1234", "192.0.2.10", http.StatusNoContent},
		{"trusted peer and denied client", "10.42.0.3:1234", "192.0.2.11", http.StatusForbidden},
		{"untrusted peer cannot spoof client", "203.0.113.5:1234", "192.0.2.10", http.StatusForbidden},
		{"trusted peer with missing client", "10.42.0.3:1234", "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/auth/check", nil)
			request.RemoteAddr = tc.remote
			request.Header.Set("CF-Connecting-IP", tc.header)
			request.SetBasicAuth("client", "authorized-test-password")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestTrustedCloudflareClientsHaveSeparateAuthenticationBudgets(t *testing.T) {
	hash, err := security.HashPassword("authorized-test-password")
	if err != nil {
		t.Fatal(err)
	}
	budget := newAttackBudget()
	auth, err := services.NewAuthServiceWithProtection(testCredentials{value: models.APICredential{
		ID: 7, Username: "client", PasswordHash: hash, Active: true,
	}}, budget, 1)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := security.NewClientIPResolver([]string{"10.42.0.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{Auth: auth, ClientIPResolver: resolver})
	for _, tc := range []struct {
		client string
		want   int
	}{
		{"192.0.2.10", http.StatusNoContent},
		{"192.0.2.11", http.StatusNoContent},
		{"192.0.2.10", http.StatusTooManyRequests},
	} {
		request := httptest.NewRequest(http.MethodGet, "/v1/auth/check", nil)
		request.RemoteAddr = "10.42.0.3:1234"
		request.Header.Set("CF-Connecting-IP", tc.client)
		request.SetBasicAuth("client", "authorized-test-password")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("client=%s status=%d want=%d", tc.client, response.Code, tc.want)
		}
	}
}
