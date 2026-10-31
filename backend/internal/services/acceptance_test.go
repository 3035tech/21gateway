package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	callbackclient "github.com/21mobile/sms-gateway/internal/integrations/callback"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
)

// No sockets: unregistered destinations fail instead of reaching a network.
func acceptanceProvider(t *testing.T) (*AccountSMSProvider, *httpmock.MockTransport, models.APICredential, string) {
	t.Helper()
	vault, err := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", 32))))
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := vault.Encrypt("acceptance-account", "acceptance-token")
	if err != nil {
		t.Fatal(err)
	}
	credential := models.APICredential{ID: 7, Username: "acceptance-account", PasswordHash: "unused-test-hash", Active: true, ProviderTokenEncrypted: encrypted, CallbacksEnabled: true, CallbackURL: "https://client.example/dlr", MOCallbacksEnabled: true, MOCallbackURL: "https://client.example/mo"}
	transport := httpmock.NewMockTransport()
	providerID := uuid.NewString()
	transport.RegisterResponder(http.MethodPost, "https://provider.example/api/send", func(r *http.Request) (*http.Response, error) {
		var body whitelabel.SendRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.Header.Get("X-API-TOKEN") != "acceptance-token" || body.Message != "Ola, acao!" || body.Destination != "5511987654321" || body.ReferenceID != "customer-ref" || body.CallbackLevel != whitelabel.CallbackNone {
			t.Fatal("incorrect provider translation/account")
		}
		return httpmock.NewJsonResponse(http.StatusOK, whitelabel.SendResponse{MessageID: providerID, Status: whitelabel.StatusSent})
	})
	client, err := whitelabel.NewWithTransport("https://provider.example", "", transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	accountProvider, err := NewAccountSMSProvider(whitelabel.NewAdapter(client, vault))
	if err != nil {
		t.Fatal(err)
	}
	return accountProvider, transport, credential, providerID
}
func acceptanceSMS() dtos.LegacySMS {
	input := validSMS()
	input.MessageText = "Olá, ação! 😊"
	return input
}

// Exercises services and real adapters; SQL/leases/deduplication require PostgreSQL below.
func TestAcceptanceSMSDLRAndMOAdapters(t *testing.T) {
	provider, transport, credential, providerID := acceptanceProvider(t)
	ctx := WithCredential(context.Background(), credential)
	attempts := &memoryAttempts{}
	result, err := NewSMSService(attempts, provider).Send(ctx, credential.ID, []dtos.LegacySMS{acceptanceSMS()})
	if err != nil || len(result) != 1 || result[0].Status != dtos.LegacyStatusOK {
		t.Fatal("SMS acceptance", result, err)
	}
	if len(attempts.completed) != 1 || attempts.completed[0].ProviderMessageID == nil || *attempts.completed[0].ProviderMessageID != providerID {
		t.Fatal("provider association lost")
	}
	vault := testMOVault(t)
	events := &moMemory{}
	input := moInput()
	input.Body.ProviderMessageID = providerID
	input.Body.CredentialID = credential.ID
	input.Body.Message = "Olá, recebi! 😊"
	if err := NewMOService(events, vault).Accept(ctx, input); err != nil {
		t.Fatal(err)
	}
	if len(events.events) != 1 || strings.Contains(events.events[0].EncryptedMessage, input.Body.Message) {
		t.Fatal("MO not protected")
	}
	event := events.events[0]
	event.ID = 11
	repo := &moWorkerMemory{job: models.MOCallbackJob{ID: 1, Attempts: 1, URL: credential.MOCallbackURL}, event: event, attempt: attempts.completed[0], allowed: true}
	calls := 0
	transport.RegisterResponder(http.MethodPost, credential.MOCallbackURL, func(r *http.Request) (*http.Response, error) {
		calls++
		var payload dtos.LegacyMOCallback
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.MTID != attempts.completed[0].ID || payload.CorrelationID != "customer-ref" || payload.Message != input.Body.Message || payload.Mobile21Identifier != event.ID || r.Header.Get("X-21Mobile-Event-ID") != "mo:11" {
			t.Fatal("MO translation lost association/content")
		}
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})
	sender := callbackclient.NewWithTransport(&http.Client{Transport: transport})
	dlrCalls := 0
	transport.RegisterResponder(http.MethodGet, credential.CallbackURL, func(r *http.Request) (*http.Response, error) {
		dlrCalls++
		if r.URL.Query().Get("idSms") != result[0].IDSMS || r.URL.Query().Get("correlationId") != "customer-ref" || r.URL.Query().Get("status") != string(models.DeliveryDelivered) || r.Header.Get("X-21Mobile-Event-ID") != "21" {
			t.Fatal("DLR translation lost association")
		}
		if dlrCalls == 1 {
			return httpmock.NewStringResponse(http.StatusServiceUnavailable, ""), nil
		}
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})
	dlrJob := models.CallbackJob{EventID: 21, URL: credential.CallbackURL, Status: models.DeliveryDelivered}
	if ok, code := sender.Deliver(ctx, dlrJob, attempts.completed[0]); ok || code != models.CallbackHTTPError {
		t.Fatal("DLR failure hidden")
	}
	if ok, code := sender.Deliver(ctx, dlrJob, attempts.completed[0]); !ok || code != "" {
		t.Fatal("DLR recovery failed")
	}
	if err := NewMOWorker(repo, sender, vault).process(ctx); err != nil {
		t.Fatal(err)
	}
	if !repo.delivered || calls != 1 || transport.GetTotalCallCount() != 4 {
		t.Fatal("incomplete SMS/DLR/MO flow")
	}
}

func TestPostgresSMSCallbackAcceptance(t *testing.T) {
	ctx, db := postgresFixture(t)
	provider, transport, credential, providerID := acceptanceProvider(t)
	credential.ID = 0
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	other := models.APICredential{Username: "other-account", PasswordHash: "unused-test-hash", Active: true}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	ctx = WithCredential(ctx, credential)
	results, err := NewSMSServiceWithLimiter(repositories.NewSMSAttempts(db), provider, repositories.NewSendLimits(db)).Send(ctx, credential.ID, []dtos.LegacySMS{acceptanceSMS()})
	if err != nil || len(results) != 1 || results[0].Status != dtos.LegacyStatusOK {
		t.Fatal("SMS acceptance", results, err)
	}
	var attempt models.SMSAttempt
	if err := db.Where("provider_message_id = ?", providerID).First(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	if results[0].IDSMS != strconv.FormatUint(attempt.ID, 10) || attempt.CredentialID != credential.ID {
		t.Fatal("SMS association")
	}
	callbacks := repositories.NewCallbacks(db)
	dlr := dtos.DeliveryInput{}
	dlr.Body.EventKey = "acceptance-dlr"
	dlr.Body.ProviderMessageID = providerID
	dlr.Body.Status = models.DeliveryDelivered
	dlr.Body.OccurredAt = time.Now().UTC().Truncate(time.Microsecond)
	for range 2 {
		if err := NewCallbackService(callbacks).Accept(ctx, dlr); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("UPDATE callback_jobs SET available_at=NOW()-INTERVAL '1 second' WHERE attempt_id=?", attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	dlrCalls := 0
	transport.RegisterResponder(http.MethodGet, credential.CallbackURL, func(r *http.Request) (*http.Response, error) {
		dlrCalls++
		if r.URL.Query().Get("idSms") != results[0].IDSMS || r.URL.Query().Get("correlationId") != "customer-ref" || r.URL.Query().Get("status") != string(models.DeliveryDelivered) {
			t.Fatal("DLR payload")
		}
		if dlrCalls == 1 {
			return httpmock.NewStringResponse(http.StatusServiceUnavailable, ""), nil
		}
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})
	sender := callbackclient.NewWithTransport(&http.Client{Transport: transport})
	dlrWorker := NewCallbackWorker(callbacks, sender)
	if err := dlrWorker.process(ctx); err != nil {
		t.Fatal(err)
	}
	var dlrJob models.CallbackJob
	if err := db.Where("attempt_id = ?", attempt.ID).First(&dlrJob).Error; err != nil {
		t.Fatal(err)
	}
	if dlrJob.State != models.CallbackPending || dlrJob.Attempts != 1 || dlrJob.LastError != models.CallbackHTTPError || !dlrJob.AvailableAt.After(time.Now()) {
		t.Fatal("DLR failure not deferred")
	}
	if err := dlrWorker.process(ctx); err != nil {
		t.Fatal(err)
	}
	if dlrCalls != 1 {
		t.Fatal("DLR retry ignored backoff")
	}
	if err := db.Model(&dlrJob).Update("available_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := dlrWorker.process(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.First(&dlrJob, dlrJob.ID).Error; err != nil {
		t.Fatal(err)
	}
	if dlrJob.State != models.CallbackDelivered || dlrJob.Attempts != 2 || dlrCalls != 2 {
		t.Fatal("DLR recovery/deduplication")
	}
	moRepo := repositories.NewMOs(db)
	vault := testMOVault(t)
	mo := NewMOService(moRepo, vault)
	input := moInput()
	input.Body.EventKey = "acceptance-mo"
	input.Body.CredentialID = credential.ID
	input.Body.ProviderMessageID = providerID
	input.Body.Message = "Resposta: Olá 😊"
	for range 2 {
		if err := mo.Accept(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("UPDATE mo_callback_jobs SET available_at=NOW()-INTERVAL '1 second' WHERE attempt_id=?", attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	cross := input
	cross.Body.CredentialID = other.ID
	assertCode(t, mo.Accept(ctx, cross), apperrors.CodeConflict)
	moCalls := 0
	transport.RegisterResponder(http.MethodPost, credential.MOCallbackURL, func(r *http.Request) (*http.Response, error) {
		moCalls++
		var payload dtos.LegacyMOCallback
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.MTID != attempt.ID || payload.Message != input.Body.Message || payload.CorrelationID != attempt.CorrelationID {
			t.Fatal("MO payload")
		}
		if moCalls == 1 {
			return httpmock.NewStringResponse(http.StatusServiceUnavailable, ""), nil
		}
		return httpmock.NewStringResponse(http.StatusNoContent, ""), nil
	})
	worker := NewMOWorker(moRepo, sender, vault)
	if err := worker.process(ctx); err != nil {
		t.Fatal(err)
	}
	var moJob models.MOCallbackJob
	if err := db.Where("attempt_id = ?", attempt.ID).First(&moJob).Error; err != nil {
		t.Fatal(err)
	}
	if moJob.State != models.CallbackPending || moJob.LastError != models.CallbackHTTPError || moJob.Attempts != 1 || !moJob.AvailableAt.After(time.Now()) {
		t.Fatal("MO failure not deferred")
	}
	if err := worker.process(ctx); err != nil {
		t.Fatal(err)
	}
	if moCalls != 1 {
		t.Fatal("MO retry ignored backoff")
	}
	if err := db.Model(&moJob).Update("available_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := worker.process(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.First(&moJob, moJob.ID).Error; err != nil {
		t.Fatal(err)
	}
	if moJob.State != models.CallbackDelivered || moJob.Attempts != 2 || moCalls != 2 {
		t.Fatal("MO recovery/deduplication")
	}
	var count int64
	if err := db.Model(&models.MOCallbackJob{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("MO duplicate jobs", count, err)
	}
	polling := repositories.NewLegacyStatuses(db)
	if rows, err := polling.Consume(ctx, other.ID, 100); err != nil || len(rows) != 0 {
		t.Fatal("DLR account isolation", err)
	}
	if rows, err := polling.Consume(ctx, credential.ID, 100); err != nil || len(rows) != 1 {
		t.Fatal("callback consumed polling", err)
	}
	if rows, err := NewLegacyMOService(moRepo, vault).Query(ctx, other.ID, input.Body.ReceivedAt.Unix()); err != nil || len(rows.ListMO) != 0 {
		t.Fatal("MO query account isolation", err)
	}
	// Old destinations must be blocked after account configuration changes.
	next := dlr
	next.Body.EventKey = "changed-url-dlr"
	next.Body.ProviderMessageID = uuid.NewString()
	nextAttempt := models.SMSAttempt{CredentialID: credential.ID, Destination: attempt.Destination, Encoding: attempt.Encoding, State: models.SubmissionAccepted, ProviderMessageID: &next.Body.ProviderMessageID}
	if err := db.Create(&nextAttempt).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewCallbackService(callbacks).Accept(ctx, next); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE callback_jobs SET available_at=NOW()-INTERVAL '1 second' WHERE attempt_id=?", nextAttempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&credential).Update("callback_url", "https://client.example/new-dlr").Error; err != nil {
		t.Fatal(err)
	}
	if err := dlrWorker.process(ctx); err != nil {
		t.Fatal(err)
	}
	var blocked models.CallbackJob
	if err := db.Where("attempt_id = ?", nextAttempt.ID).First(&blocked).Error; err != nil {
		t.Fatal(err)
	}
	if blocked.State != models.CallbackFailed || blocked.LastError != models.CallbackAccessBlocked || dlrCalls != 2 {
		t.Fatal("old DLR destination used")
	}
	// MO uses the same current-destination policy.
	nextMO := input
	nextMO.Body.EventKey = "changed-url-mo"
	if err := mo.Accept(ctx, nextMO); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE mo_callback_jobs SET available_at=NOW()-INTERVAL '1 second' WHERE attempt_id=? AND state=?", attempt.ID, models.CallbackPending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&credential).Update("mo_callback_url", "https://client.example/new-mo").Error; err != nil {
		t.Fatal(err)
	}
	if err := worker.process(ctx); err != nil {
		t.Fatal(err)
	}
	var blockedMO models.MOCallbackJob
	if err := db.Where("event_id = (SELECT id FROM mo_events WHERE credential_id = ? AND event_key = ?)", credential.ID, nextMO.Body.EventKey).First(&blockedMO).Error; err != nil {
		t.Fatal(err)
	}
	if blockedMO.State != models.CallbackFailed || blockedMO.LastError != models.CallbackAccessBlocked || moCalls != 2 {
		t.Fatal("old MO destination used")
	}
	if transport.GetTotalCallCount() != 5 {
		t.Fatal("unexpected send/callback calls", transport.GetTotalCallCount())
	}
}
