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

func (r *pollingFixture) ByDate(_ context.Context, id uint64, start, end time.Time, correlationID, destination string, limit int) ([]models.SMSAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id != 7 || limit != dtos.LegacyStatusPageSize+1 {
		return nil, errors.New("incorrect scope")
	}
	var out []models.SMSAttempt
	for _, row := range r.rows {
		if correlationID != "" && row.CorrelationID != correlationID {
			continue
		}
		if destination != "" && row.Destination != destination {
			continue
		}
		if !row.CreatedAt.Before(end) || row.CreatedAt.Before(start) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
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
	router := New(nil, true, func(context.Context) error { return nil }, Dependencies{Auth: auth, V1Statuses: services.NewV1StatusService(repo)})
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

func TestV1StatusByDateHTTP(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true, WhitelistEnabled: true, AllowedIPs: []string{"192.0.2.0/24"}}})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 14, 15, 24, 21, 0, time.UTC)
	repo := &pollingFixture{rows: []models.SMSAttempt{
		{CorrelationID: "ref-a", Destination: "5511999999999", CreatedAt: at, DeliveryStatus: string(models.DeliveryDelivered)},
		{CorrelationID: "ref-b", Destination: "5511888888888", CreatedAt: at, DeliveryStatus: string(models.DeliverySent)},
	}}
	router := New(nil, true, func(context.Context) error { return nil }, Dependencies{Auth: auth, V1Statuses: services.NewV1StatusService(repo)})
	request := func(query, password, peer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/v1/status/date"+query, nil)
		req.RemoteAddr = peer
		req.SetBasicAuth("client", password)
		req.Header.Set("X-Forwarded-For", "192.0.2.1")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	// auth guard
	if res := request("?date=1791991461", "wrong", "192.0.2.1:80"); res.Code != http.StatusUnauthorized {
		t.Fatal("unauthorized", res.Code)
	}
	if res := request("?date=1791991461", "test-password-only", "198.51.100.1:80"); res.Code != http.StatusForbidden {
		t.Fatal("IP bypass", res.Code)
	}
	// date required
	if res := request("", "test-password-only", "192.0.2.1:80"); res.Code != http.StatusUnprocessableEntity {
		t.Fatal("missing date", res.Code, res.Body.String())
	}
	// all results for the day
	res := request("?date=1791991461", "test-password-only", "192.0.2.1:80")
	var body dtos.LegacyStatusBody
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil || body.Code != dtos.LegacyCodeAccepted || len(body.SentStatus) != 2 {
		t.Fatal(res.Code, res.Body.String())
	}
	// non-consuming: same result on repeat
	res2 := request("?date=1791991461", "test-password-only", "192.0.2.1:80")
	var body2 dtos.LegacyStatusBody
	if res2.Code != http.StatusOK || json.Unmarshal(res2.Body.Bytes(), &body2) != nil || len(body2.SentStatus) != 2 {
		t.Fatal("non-consuming", res2.Body.String())
	}
	// filter by correlationid
	res = request("?date=1791991461&correlationid=ref-a", "test-password-only", "192.0.2.1:80")
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil || len(body.SentStatus) != 1 || body.SentStatus[0].CorrelationID != "ref-a" {
		t.Fatal("correlationid filter", res.Code, res.Body.String())
	}
	// filter by destination
	res = request("?date=1791991461&destination=5511888888888", "test-password-only", "192.0.2.1:80")
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil || len(body.SentStatus) != 1 || body.SentStatus[0].Destination != "5511888888888" {
		t.Fatal("destination filter", res.Code, res.Body.String())
	}
	// The official examples use a trailing slash; both forms are accepted.
	slashRequest := httptest.NewRequest(http.MethodGet, "/v1/status/date/?date=1791991461", nil)
	slashRequest.RemoteAddr = "192.0.2.1:80"
	slashRequest.SetBasicAuth("client", "test-password-only")
	slashRequest.Header.Set("X-Forwarded-For", "192.0.2.1")
	slashResponse := httptest.NewRecorder()
	router.ServeHTTP(slashResponse, slashRequest)
	if slashResponse.Code != http.StatusOK {
		t.Fatal("trailing slash route", slashResponse.Code, slashResponse.Body.String())
	}
	// empty day
	res = request("?date=1700000000", "test-password-only", "192.0.2.1:80")
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil || body.Code != dtos.LegacyCodeNoRecords || len(body.SentStatus) != 0 {
		t.Fatal("empty day", res.Code, res.Body.String())
	}
	// Until continuation semantics are confirmed, do not silently return a partial list.
	repo.rows = make([]models.SMSAttempt, dtos.LegacyStatusPageSize+1)
	for i := range repo.rows {
		repo.rows[i] = models.SMSAttempt{CorrelationID: "overflow", Destination: "5511999999999", CreatedAt: at, DeliveryStatus: string(models.DeliveryDelivered)}
	}
	res = request("?date=1791991461", "test-password-only", "192.0.2.1:80")
	if res.Code != http.StatusServiceUnavailable {
		t.Fatal("overflow should not be silently truncated", res.Code, res.Body.String())
	}
}
