package httpapi

import (
	"context"
	"encoding/base64"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type moIngressMemory struct{ events []models.MOEvent }

func (r *moIngressMemory) Accept(_ context.Context, e models.MOEvent) error {
	r.events = append(r.events, e)
	return nil
}
func TestMOInternalIngressOptInAndAuthentication(t *testing.T) {
	vault, _ := security.NewMOContentVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	token := strings.Repeat("t", 32)
	body := `{"eventKey":"mo-1","credentialId":7,"providerMessageId":"550e8400-e29b-41d4-a716-446655440000","phoneNumber":"5511998877665","message":"private MO content","receivedAt":"2026-10-14T12:00:00Z"}`
	for _, tc := range []struct {
		name          string
		enabled       bool
		authorization string
		want          int
		persist       bool
	}{
		{"disabled", false, "Bearer " + token, http.StatusNotFound, false}, {"unauthorized", true, "", http.StatusUnauthorized, false}, {"wrong token", true, "Bearer wrong", http.StatusUnauthorized, false}, {"accepted", true, "Bearer " + token, http.StatusNoContent, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &moIngressMemory{}
			router := New(nil, true, func(context.Context) error { return nil }, Dependencies{MO: services.NewMOService(repo, vault), MOIngressEnabled: tc.enabled, CallbackIngressToken: token})
			req := httptest.NewRequest(http.MethodPost, "/internal/mo", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", tc.authorization)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.want || (len(repo.events) == 1) != tc.persist {
				t.Fatal(res.Code, res.Body.String())
			}
			if tc.persist && strings.Contains(repo.events[0].EncryptedMessage, "private MO content") {
				t.Fatal("plaintext persisted")
			}
		})
	}
}
