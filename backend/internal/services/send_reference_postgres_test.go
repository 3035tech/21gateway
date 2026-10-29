package services

import (
	"strconv"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

func TestPostgresSendReferencePublicationAndDuplicateDelivery(t *testing.T) {
	ctx, db := postgresFixture(t)
	credential := models.APICredential{Username: "send-reference", PasswordHash: "unused", Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	scheduled := repositories.NewScheduledSMS(db)
	refs := repositories.NewSendReferences(db)
	vault := scheduledVault(t)
	inputs := []dtos.LegacySMS{validSMS(), futureSMS()}
	results, err := NewScheduledSMSService(scheduled, vault).EnqueueAll(ctx, credential.ID, inputs)
	if err != nil || len(results) != 2 {
		t.Fatal("admission", err)
	}
	// The database clock can lag the process clock by a few milliseconds in
	// containerized tests. Make the immediate fixture due by database time.
	if err := db.Exec("UPDATE scheduled_sms SET due_at=NOW()-INTERVAL '1 second', available_at=NOW()-INTERVAL '1 second' WHERE send_reference=TRUE AND attempt_id=?", results[0].IDSMS).Error; err != nil {
		t.Fatal(err)
	}
	first, err := refs.ClaimPublication(ctx)
	if err != nil || first == nil || strconv.FormatUint(first.AttemptID, 10) != results[0].IDSMS {
		t.Fatal("one immediate ID publication", err)
	}
	if duplicate, err := refs.ClaimPublication(ctx); err != nil || duplicate != nil {
		t.Fatal("future or leased send published", err)
	}
	if err := refs.PublicationResult(ctx, *first, false, time.Minute); err != nil {
		t.Fatal("failed publication result", err)
	}
	if next, err := refs.ClaimPublication(ctx); err != nil || next != nil {
		t.Fatal("failed publication retried without backoff", err)
	}
	if err := db.Model(&models.ScheduledSMS{}).Where("id=?", first.ID).Update("publish_after", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	first, err = refs.ClaimPublication(ctx)
	if err != nil || first == nil || first.PublishAttempts != 2 {
		t.Fatal("failed reference not retried", err)
	}
	if err := refs.PublicationResult(ctx, *first, true, time.Second); err != nil {
		t.Fatal(err)
	}
	job, done, ready, err := refs.Claim(ctx, first.AttemptID)
	if err != nil || done || !ready || job.Attempt.ID != first.AttemptID {
		t.Fatal("send claim", err)
	}
	if _, done, ready, err := refs.Claim(ctx, first.AttemptID); err != nil || done || ready {
		t.Fatal("duplicate claimed", err)
	}
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440001", Status: whitelabel.StatusSent}}
	worker := NewScheduledSMSWorker(scheduled, provider, &countLimiter{allowed: true}, vault)
	if progress, err := worker.processGroup(ctx, []models.ScheduledSMS{job}); err != nil || !progress {
		t.Fatal("worker send", err)
	}
	if len(provider.inputs) != 1 {
		t.Fatal("provider called more than once")
	}
	if _, done, ready, err := refs.Claim(ctx, first.AttemptID); err != nil || !done || ready {
		t.Fatal("terminal duplicate not acknowledged", err)
	}
	var attempt models.SMSAttempt
	if err := db.First(&attempt, first.AttemptID).Error; err != nil || attempt.State != models.SubmissionAccepted {
		t.Fatal("provider outcome missing", err)
	}
	var stored models.ScheduledSMS
	if err := db.First(&stored, first.ID).Error; err != nil || stored.EncryptedMessage != "" || stored.State != models.ScheduleDone {
		t.Fatal("terminal content retained", err)
	}
}
