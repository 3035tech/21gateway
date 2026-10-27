package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"testing"
	"time"
)

type moQueryMemory struct {
	rows         []models.MOEvent
	credentialID uint64
	start, end   time.Time
	limit        int
	calls        int
}

func (r *moQueryMemory) QueryDay(_ context.Context, id uint64, start, end time.Time, limit int) ([]models.MOEvent, error) {
	r.calls++
	r.credentialID = id
	r.start = start
	r.end = end
	r.limit = limit
	return r.rows, nil
}
func TestLegacyMOQueryDecryptsOnlyAccountAndPreservesResponse(t *testing.T) {
	vault := testMOVault(t)
	encrypted, err := vault.Encrypt(7, "event", "Olá\n😊")
	if err != nil {
		t.Fatal(err)
	}
	attempt := uint64(42)
	at := time.Date(2026, 10, 14, 15, 5, 1, 0, time.UTC)
	repo := &moQueryMemory{rows: []models.MOEvent{{ID: 1795, CredentialID: 7, EventKey: "event", AttemptID: &attempt, EncryptedMessage: encrypted, PhoneNumber: "5511998877665", ReceivedAt: at}}}
	service := NewLegacyMOService(repo, vault)
	for range 2 {
		body, err := service.Query(context.Background(), 7, at.Unix())
		if err != nil {
			t.Fatal(err)
		}
		if body.Code != dtos.LegacyCodeAccepted || len(body.ListMO) != 1 || body.ListMO[0].Message != "Olá\n😊" || body.ListMO[0].IDMO != "1795" || body.ListMO[0].IDMT != "42" || body.ListMO[0].ReceivedDate != "2026-10-14 12:05:01-03" {
			t.Fatalf("wrong response %+v", body)
		}
	}
	if repo.credentialID != 7 || repo.start != time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC) || repo.end != time.Date(2026, 10, 15, 3, 0, 0, 0, time.UTC) || repo.limit != MaximumLegacyMORecords+1 || repo.calls != 2 {
		t.Fatal("incorrect query bounds", repo.start, repo.end)
	}
	repo.rows[0].CredentialID = 8
	_, err = service.Query(context.Background(), 7, at.Unix())
	assertCode(t, err, apperrors.CodeServiceUnavailable)
	repo.rows[0].CredentialID = 7
	repo.rows[0].EncryptedMessage = "corrupt"
	body, err := service.Query(context.Background(), 7, at.Unix())
	assertCode(t, err, apperrors.CodeServiceUnavailable)
	if len(body.ListMO) != 0 {
		t.Fatal("partial content returned after corruption")
	}
}
func TestLegacyMOQueryDoesNotSilentlyTruncate(t *testing.T) {
	repo := &moQueryMemory{rows: make([]models.MOEvent, MaximumLegacyMORecords+1)}
	_, err := NewLegacyMOService(repo, testMOVault(t)).Query(context.Background(), 7, time.Now().Unix())
	assertCode(t, err, apperrors.CodeServiceUnavailable)
	repo.rows = nil
	body, err := NewLegacyMOService(repo, nil).Query(context.Background(), 7, time.Now().Unix())
	if err != nil || body.Code != dtos.LegacyCodeNoRecords || body.ListMO == nil || len(body.ListMO) != 0 {
		t.Fatal("incorrect empty response", err)
	}
	for _, timestamp := range []int64{-1, MaximumLegacyQueryTimestamp + 1} {
		_, err := NewLegacyMOService(repo, nil).Query(context.Background(), 7, timestamp)
		assertCode(t, err, apperrors.CodeInvalidRequest)
	}
}
func TestLegacyDayRangeCalendarTransitions(t *testing.T) {
	for _, tc := range []struct {
		date  string
		hours time.Duration
	}{{"2018-11-04T12:00:00-02:00", 23}, {"2019-02-16T12:00:00-02:00", 25}, {"2026-10-14T12:00:00-03:00", 24}} {
		at, err := time.Parse(time.RFC3339, tc.date)
		if err != nil {
			t.Fatal(err)
		}
		start, end := legacyDayRange(at.Unix())
		if end.Sub(start) != tc.hours*time.Hour || at.Before(start) || !at.Before(end) {
			t.Fatal("wrong calendar bounds", tc.date, start, end)
		}
		y, m, d := at.In(legacyQueryLocation).Date()
		sy, sm, sd := start.In(legacyQueryLocation).Date()
		if y != sy || m != sm || d != sd {
			t.Fatal("day starts on previous date", tc.date, start)
		}
	}
}
