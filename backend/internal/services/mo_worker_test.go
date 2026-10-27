package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/models"
	"testing"
	"time"
)

type moWorkerMemory struct {
	job       models.MOCallbackJob
	event     models.MOEvent
	attempt   models.SMSAttempt
	allowed   bool
	finished  bool
	delivered bool
	code      string
	detached  bool
}

func (r *moWorkerMemory) Reconcile(context.Context) error                      { return nil }
func (r *moWorkerMemory) Claim(context.Context) (*models.MOCallbackJob, error) { return &r.job, nil }
func (r *moWorkerMemory) DeliveryData(context.Context, models.MOCallbackJob) (models.MOEvent, models.SMSAttempt, bool, error) {
	return r.event, r.attempt, r.allowed, nil
}
func (r *moWorkerMemory) Finish(ctx context.Context, _ models.MOCallbackJob, ok bool, code string, _ time.Duration) error {
	r.finished = true
	r.delivered = ok
	r.code = code
	r.detached = ctx.Err() == nil
	return nil
}

type moWorkerSender struct {
	calls   int
	message models.MOMessage
	cancel  context.CancelFunc
	success bool
}

func (s *moWorkerSender) DeliverMO(_ context.Context, _ string, m models.MOMessage) (bool, string) {
	s.calls++
	s.message = m
	if s.cancel != nil {
		s.cancel()
	}
	if s.success {
		return true, ""
	}
	return false, models.CallbackHTTPError
}
func TestMOWorkerAuthorizationDecryptAndDetachedCompletion(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		allowed, corrupt, success bool
		wantCalls                 int
		wantCode                  string
	}{
		{"delivered", true, false, true, 1, ""}, {"HTTP failure", true, false, false, 1, models.CallbackHTTPError}, {"blocked", false, false, false, 0, models.CallbackAccessBlocked}, {"corrupt content", true, true, false, 0, models.CallbackContentUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vault := testMOVault(t)
			encrypted, err := vault.Encrypt(7, "event", "Olá\n😊")
			if err != nil {
				t.Fatal(err)
			}
			if tc.corrupt {
				encrypted = "broken"
			}
			repo := &moWorkerMemory{job: models.MOCallbackJob{ID: 1, Attempts: 1, URL: "https://client.example/mo"}, event: models.MOEvent{ID: 2, CredentialID: 7, EventKey: "event", EncryptedMessage: encrypted, PhoneNumber: "5511998877665", ReceivedAt: time.Now()}, attempt: models.SMSAttempt{ID: 42, CorrelationID: "ref"}, allowed: tc.allowed}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sender := &moWorkerSender{cancel: cancel, success: tc.success}
			if err := NewMOWorker(repo, sender, vault).process(ctx); err != nil {
				t.Fatal(err)
			}
			if sender.calls != tc.wantCalls || !repo.finished || repo.code != tc.wantCode || repo.delivered != tc.success || !repo.detached {
				t.Fatal("wrong worker outcome", sender.calls, repo.code)
			}
			if sender.calls == 1 && (sender.message.ID != 2 || sender.message.AttemptID != 42 || sender.message.CorrelationID != "ref" || sender.message.Message != "Olá\n😊") {
				t.Fatal("incorrect resolved message")
			}
		})
	}
}
