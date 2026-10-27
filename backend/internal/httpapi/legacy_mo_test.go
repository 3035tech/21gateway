package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type legacyMOFixture struct {
	rows  []models.MOEvent
	calls int
}

func (r *legacyMOFixture) QueryDay(_ context.Context, id uint64, start, end time.Time, limit int) ([]models.MOEvent, error) {
	r.calls++
	if id != 7 {
		panic("account scope incorrect")
	}
	return r.rows, nil
}
func TestLegacyMOHTTPContractAndAuthorization(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", Active: true, PasswordHash: hash, WhitelistEnabled: true, AllowedIPs: []string{"192.0.2.0/24"}}})
	if err != nil {
		t.Fatal(err)
	}
	vault, err := security.NewMOContentVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := vault.Encrypt(7, "event", "Foo bar")
	if err != nil {
		t.Fatal(err)
	}
	mt := uint64(6652)
	repo := &legacyMOFixture{rows: []models.MOEvent{{ID: 1795, CredentialID: 7, AttemptID: &mt, EventKey: "event", EncryptedMessage: encrypted, PhoneNumber: "5551998877665", ReceivedAt: time.Date(2017, 4, 18, 19, 5, 1, 0, time.UTC)}}}
	router := New(nil, true, func(context.Context) error { return nil }, Dependencies{Auth: auth, LegacyMO: services.NewLegacyMOService(repo, vault)})
	request := func(query, password, remote string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/consultmo"+query, nil)
		req.RemoteAddr = remote
		req.SetBasicAuth("client", password)
		req.Header.Set("X-Forwarded-For", "192.0.2.1")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	if res := request("?date=1492538809", "wrong", "192.0.2.1:80"); res.Code != http.StatusUnauthorized || repo.calls != 0 {
		t.Fatal("auth failure", res.Code)
	}
	if res := request("?date=1492538809", "test-password-only", "198.51.100.1:80"); res.Code != http.StatusForbidden || repo.calls != 0 {
		t.Fatal("IP bypass", res.Code)
	}
	for _, query := range []string{"", "?date=abc", "?date=-1", "?date=253402300800"} {
		if res := request(query, "test-password-only", "192.0.2.1:80"); res.Code != http.StatusUnprocessableEntity || repo.calls != 0 {
			t.Fatal("invalid date reached repository", query, res.Code, res.Body.String())
		}
	}
	for range 2 {
		res := request("?date=1492538809", "test-password-only", "192.0.2.1:80")
		if res.Code != http.StatusOK {
			t.Fatal(res.Code, res.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		items := body["list_mo"].([]any)
		item := items[0].(map[string]any)
		if item["fromAdress"] != "5551998877665" || item["idMo"] != "1795" || item["idMt"] != "6652" || item["msg"] != "Foo bar" || item["receivedDate"] != "2017-04-18 16:05:01-03" || len(item) != 6 {
			t.Fatal("legacy field or type mismatch", body)
		}
		if strings.Contains(res.Body.String(), encrypted) || strings.Contains(res.Body.String(), "fingerprint") {
			t.Fatal("cipher metadata leaked")
		}
	}
	repo.rows = nil
	res := request("?date=1492538809", "test-password-only", "192.0.2.1:80")
	if !strings.Contains(res.Body.String(), `"code":6`) || !strings.Contains(res.Body.String(), `"list_mo":[]`) {
		t.Fatal("empty result", res.Body.String())
	}
}
