package services

import (
	"github.com/21mobile/sms-gateway/internal/dtos"
	"strconv"
	"testing"
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

func TestPostgresScheduledQueueLeasesAndUnknownRecovery(t *testing.T) {
	ctx, db := postgresFixture(t)
	credential := models.APICredential{Username: "scheduled-client", PasswordHash: "unused", Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	repo := repositories.NewScheduledSMS(db)
	vault := scheduledVault(t)
	input := futureSMS()
	input.CorrelationID = "schedule-ref"
	receipt, err := NewScheduledSMSService(repo, vault).Enqueue(ctx, credential.ID, []dtos.LegacySMS{input})
	if err != nil || len(receipt) != 1 {
		t.Fatal(err)
	}
	if jobs, err := repo.Claim(ctx); err != nil || len(jobs) != 0 {
		t.Fatal("future schedule claimed", err)
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("credential_id=?", credential.ID).Updates(map[string]any{"due_at": time.Now().Add(-time.Second), "available_at": time.Now().Add(-time.Second)}).Error; err != nil {
		t.Fatal(err)
	}
	jobs, err := repo.Claim(ctx)
	if err != nil || len(jobs) != 1 || strconv.FormatUint(jobs[0].AttemptID, 10) != receipt[0].IDSMS {
		t.Fatal("claim/ID", err)
	}
	if duplicate, err := repo.Claim(ctx); err != nil || len(duplicate) != 0 {
		t.Fatal("active lease claimed twice", err)
	}
	if err := repo.Release(ctx, jobs); err != nil {
		t.Fatal(err)
	}
	if err := repo.Begin(ctx, jobs); err == nil {
		t.Fatal("stale lease dispatched")
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("id=?", jobs[0].ID).Update("available_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	jobs, err = repo.Claim(ctx)
	if err != nil || len(jobs) != 1 {
		t.Fatal("deferred reclaim", err)
	}
	if err := repo.Begin(ctx, jobs); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("id=?", jobs[0].ID).Update("lease_until", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if replay, err := repo.Claim(ctx); err != nil || len(replay) != 0 {
		t.Fatal("unknown dispatch replayed", err)
	}
	var job models.ScheduledSMS
	var attempt models.SMSAttempt
	if err := db.First(&job, jobs[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&attempt, jobs[0].AttemptID).Error; err != nil {
		t.Fatal(err)
	}
	if job.State != models.ScheduleUnknown || job.EncryptedMessage != "" || attempt.State != models.SubmissionUnknown || attempt.FailureCode != apperrors.CodeUnknownOutcome || attempt.DispatchedAt == nil {
		t.Fatal("unknown recovery/privacy")
	}
	if err := repo.Finish(ctx, jobs, []models.SMSAttempt{attempt}); err == nil {
		t.Fatal("stale completion overwrote unknown")
	}
	// Successful execution keeps the accepted ID and commits UUID/outbox state together.
	second, err := NewScheduledSMSService(repo, vault).Enqueue(ctx, credential.ID, []dtos.LegacySMS{input})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("state=?", models.ScheduleQueued).Updates(map[string]any{"due_at": time.Now().Add(-time.Second), "available_at": time.Now().Add(-time.Second)}).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubProvider{enabled: true}
	provider.response.MessageID = "550e8400-e29b-41d4-a716-446655440099"
	if err := NewScheduledSMSWorker(repo, provider, &countLimiter{allowed: true}, vault).process(ctx); err != nil {
		t.Fatal(err)
	}
	if len(provider.inputs) != 1 || provider.inputs[0].Index != second[0].IDSMS {
		t.Fatal("receipt ID replaced")
	}
	// A fresh lookup avoids GORM's previous primary-key predicate.
	var finished models.ScheduledSMS
	if err := db.Where("attempt_id=?", second[0].IDSMS).First(&finished).Error; err != nil {
		t.Fatal(err)
	}
	if finished.State != models.ScheduleDone || finished.EncryptedMessage != "" {
		t.Fatal("completed content not purged")
	}
	// Admission and dispatch contend for the same account request budget.
	limits := repositories.NewSendLimits(db)
	if allowed, err := limits.Allow(ctx, credential.ID, 0); err != nil || !allowed {
		t.Fatal("schedule request admission", err)
	}
	if allowed, err := limits.Allow(ctx, credential.ID, 1); err != nil || allowed {
		t.Fatal("account budget bypass", err)
	}
	if allowed, err := repositories.NewDispatchSendLimits(db).Allow(ctx, credential.ID, 1); err != nil || !allowed {
		t.Fatal("queue admission starved dispatch", err)
	}

}
