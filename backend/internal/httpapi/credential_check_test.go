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
