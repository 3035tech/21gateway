package services

import (
	"sync"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/google/uuid"
)

func TestPostgresDispatchOutboxPublicationLeasesAndPartialBudget(t *testing.T) {
	ctx, db := postgresFixture(t)
	credentials := []models.APICredential{{Username: "dispatch-a", PasswordHash: "unused", Active: true}, {Username: "dispatch-b", PasswordHash: "unused", Active: true}}
	if err := db.Create(&credentials).Error; err != nil {
		t.Fatal(err)
	}
	scheduled := repositories.NewScheduledSMS(db)
	batches := repositories.NewDispatchBatches(db)
	vault := scheduledVault(t)
	messages := make([]dtos.LegacySMS, 200)
	for i := range messages {
		messages[i] = futureSMS()
	}
	if _, err := NewScheduledSMSService(scheduled, vault).Enqueue(ctx, credentials[0].ID, messages); err != nil {
		t.Fatal(err)
	}
	if _, err := NewScheduledSMSService(scheduled, vault).Enqueue(ctx, credentials[1].ID, []dtos.LegacySMS{futureSMS()}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("state=?", models.ScheduleQueued).Updates(map[string]any{"due_at": time.Now().Add(-time.Second), "available_at": time.Now().Add(-time.Second)}).Error; err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := batches.Prepare(ctx); err != nil {
				failures <- err
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	if err := batches.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	var all []models.DispatchBatch
	if err := db.Find(&all).Error; err != nil || len(all) != 2 {
		t.Fatal("one open batch per account", len(all), err)
	}
	for _, batch := range all {
		var count int64
		if err := db.Model(&models.ScheduledSMS{}).Where("dispatch_batch_id=?", batch.ID).Count(&count).Error; err != nil || count == 0 || count > 100 {
			t.Fatal("fair bounded batch", count, err)
		}
	}
	// Colima's database clock can lag the Go process by a few milliseconds.
	// Make the newly created publication fixtures due by database time.
	if err := db.Exec("UPDATE sms_dispatch_batches SET available_at=NOW()-INTERVAL '1 second' WHERE published_at IS NULL").Error; err != nil {
		t.Fatal(err)
	}
	publication, err := batches.ClaimPublication(ctx)
	if err != nil || publication == nil {
		t.Fatal("outbox publication missing", err)
	}
	if err := batches.PublicationResult(ctx, *publication, false, time.Minute); err != nil {
		t.Fatal(err)
	}
	var failed models.DispatchBatch
	if err := db.First(&failed, "id=?", publication.ID).Error; err != nil || failed.PublishedAt != nil || !failed.AvailableAt.After(time.Now()) {
		t.Fatal("failed publication lost intent", err)
	}
	if err := db.Model(&failed).Update("available_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	retry, err := batches.ClaimPublication(ctx)
	if err != nil || retry == nil || retry.ID != publication.ID || *retry.LeaseToken == *publication.LeaseToken {
		t.Fatal("publication retry changed batch or lease", err)
	}
	if err := batches.PublicationResult(ctx, *retry, true, 0); err != nil {
		t.Fatal(err)
	}
	jobs, done, err := batches.ClaimBatch(ctx, retry.ID)
	if err != nil || done || len(jobs) == 0 {
		t.Fatal("SQS reference not claimable", err)
	}
	if duplicate, done, err := batches.ClaimBatch(ctx, retry.ID); err != nil || done || len(duplicate) != 0 {
		t.Fatal("concurrent duplicate claimed", err)
	}
	if err := scheduled.Begin(ctx, jobs); err != nil {
		t.Fatal(err)
	}
	attempts := make([]models.SMSAttempt, len(jobs))
	for i, job := range jobs {
		attempts[i] = job.Attempt
		attempts[i].State = models.SubmissionAccepted
		id := uuid.NewString()
		attempts[i].ProviderMessageID = &id
	}
	if err := scheduled.Finish(ctx, jobs, attempts); err != nil {
		t.Fatal(err)
	}
	if done, err := batches.Complete(ctx, retry.ID); err != nil || !done {
		t.Fatal("completed batch not durable", err)
	}
	if duplicate, done, err := batches.ClaimBatch(ctx, retry.ID); err != nil || !done || len(duplicate) != 0 {
		t.Fatal("completed reference replayed", err)
	}
	if err := batches.PublicationResult(ctx, *retry, true, 0); err != nil {
		t.Fatal("completion raced publisher acknowledgment", err)
	}
	// A lost in-flight response becomes unknown; repeated references never resend it.
	var other models.DispatchBatch
	if err := db.Where("completed_at IS NULL").First(&other).Error; err != nil {
		t.Fatal(err)
	}
	inflight, _, err := batches.ClaimBatch(ctx, other.ID)
	if err != nil || len(inflight) == 0 {
		t.Fatal(err)
	}
	if err := scheduled.Begin(ctx, inflight); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("dispatch_batch_id=?", other.ID).Update("lease_until", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if replay, done, err := batches.ClaimBatch(ctx, other.ID); err != nil || !done || len(replay) != 0 {
		t.Fatal("expired network call replayed", err)
	}
	var unknown int64
	if err := db.Model(&models.ScheduledSMS{}).Where("dispatch_batch_id=? AND state=? AND encrypted_message=''", other.ID, models.ScheduleUnknown).Count(&unknown).Error; err != nil || unknown != int64(len(inflight)) {
		t.Fatal("unknown content/state recovery", err)
	}
	// Keep a deterministic reservation visible throughout this test without sleeps.
	if err := db.Exec("INSERT INTO send_reservations(credential_id,messages,created_at,scope) VALUES (?,199,NOW()+INTERVAL '1 minute','dispatch')", credentials[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	granted, err := repositories.NewDispatchSendLimits(db).ReserveUpTo(ctx, credentials[1].ID, 200)
	if err != nil || granted != 1 {
		t.Fatal("large batch starved despite available capacity", granted, err)
	}
}
