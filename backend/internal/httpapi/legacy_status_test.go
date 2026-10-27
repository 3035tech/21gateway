package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type pollingFixture struct {
	mu    sync.Mutex
	rows  []models.SMSAttempt
	fail  bool
	calls int
}

func (r *pollingFixture) Consume(_ context.Context, id uint64, limit int) ([]models.SMSAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.fail {
		return nil, errors.New("private database information")
	}
	if id != 7 || limit != dtos.LegacyStatusPageSize {
		return nil, errors.New("incorrect scope")
	}
	rows := r.rows
	r.rows = nil
	return rows, nil
}
func TestLegacyStatusHTTP(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true, WhitelistEnabled: true, AllowedIPs: []string{"192.0.2.0/24"}}})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 14, 15, 24, 21, 0, time.UTC)
	repo := &pollingFixture{rows: []models.SMSAttempt{{CorrelationID: "ref", Destination: "5511999999999", CreatedAt: at, DeliveryStatus: string(models.DeliveryDelivered)}}}
	router := New(nil, true, func(context.Context) error { return nil }, Dependencies{Auth: auth, LegacyStatuses: services.NewLegacyStatusService(repo)})
	request := func(password, peer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
		req.RemoteAddr = peer
		req.SetBasicAuth("client", password)
		req.Header.Set("X-Forwarded-For", "192.0.2.1")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	if res := request("wrong", "192.0.2.1:80"); res.Code != http.StatusUnauthorized || repo.calls != 0 {
		t.Fatal("unauthorized polling", res.Code)
	}
	if res := request("test-password-only", "198.51.100.1:80"); res.Code != http.StatusForbidden || repo.calls != 0 {
		t.Fatal("IP bypass", res.Code)
	}
	res := request("test-password-only", "192.0.2.1:80")
	var body dtos.LegacyStatusBody
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil || body.Code != dtos.LegacyCodeAccepted || len(body.SentStatus) != 1 {
		t.Fatal(res.Code, res.Body.String())
	}
	if body.SentStatus[0].SentDate != "2026-10-14 12:24:21-03" || body.SentStatus[0].UnixSentDate != "1791991461" {
		t.Fatalf("wrong dates %+v", body.SentStatus[0])
	}
	res = request("test-password-only", "192.0.2.1:80")
	if json.Unmarshal(res.Body.Bytes(), &body) != nil || body.Code != dtos.LegacyCodeNoRecords || len(body.SentStatus) != 0 {
		t.Fatal(res.Body.String())
	}
	repo.fail = true
	res = request("test-password-only", "192.0.2.1:80")
	if res.Code != http.StatusServiceUnavailable {
		t.Fatal(res.Code, res.Body.String())
	}
}
