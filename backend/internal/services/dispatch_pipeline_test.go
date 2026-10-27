package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/dispatchqueue"
	"github.com/21mobile/sms-gateway/internal/models"
)

const pipelineBatchID = "550e8400-e29b-41d4-a716-446655440001"

type pipelineMemory struct {
	mu                     sync.Mutex
	jobs                   []models.ScheduledSMS
	leased, terminal, done bool
	moreWork               bool
	began                  int
	finishErr              error
	prepareErr             error
	batch                  *models.DispatchBatch
	publications           []bool
}

func (r *pipelineMemory) Prepare(context.Context) error { return r.prepareErr }
func (r *pipelineMemory) ClaimPublication(context.Context) (*models.DispatchBatch, error) {
	return r.batch, nil
}
func (r *pipelineMemory) PublicationResult(_ context.Context, _ models.DispatchBatch, ok bool, _ time.Duration) error {
	r.publications = append(r.publications, ok)
	return nil
}
func (r *pipelineMemory) ClaimBatch(context.Context, string) ([]models.ScheduledSMS, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done || r.terminal {
		r.done = true
		return nil, true, nil
	}
	if r.leased {
		return nil, false, nil
	}
	r.leased = true
	return r.jobs, false, nil
}
func (r *pipelineMemory) Complete(context.Context, string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done = r.terminal && !r.moreWork
	return r.done, nil
}
func (r *pipelineMemory) Claim(context.Context) ([]models.ScheduledSMS, error) { return nil, nil }
func (r *pipelineMemory) Credential(context.Context, uint64) (models.APICredential, error) {
	return models.APICredential{ID: 7, Active: true}, nil
}
func (r *pipelineMemory) Begin(context.Context, []models.ScheduledSMS) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.began++
	return nil
}
func (r *pipelineMemory) Release(context.Context, []models.ScheduledSMS) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.leased = false
	return nil
}
func (r *pipelineMemory) Reject(context.Context, []models.ScheduledSMS, apperrors.Code) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.terminal = true
	return nil
}
func (r *pipelineMemory) Finish(context.Context, []models.ScheduledSMS, []models.SMSAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finishErr != nil {
		return r.finishErr
	}
	r.terminal = true
	return nil
}

type pipelineQueue struct {
	mu                      sync.Mutex
	acks, defers, published int
	publishErr              error
	ackFailOnce             bool
	polling                 atomic.Int32
	peak                    atomic.Int32
	started                 chan struct{}
}

func (q *pipelineQueue) Publish(context.Context, string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.published++
	return q.publishErr
}
func (q *pipelineQueue) Receive(ctx context.Context) ([]dispatchqueue.Delivery, error) {
	current := q.polling.Add(1)
	defer q.polling.Add(-1)
	for old := q.peak.Load(); current > old; old = q.peak.Load() {
		if q.peak.CompareAndSwap(old, current) {
			break
		}
	}
	if q.started != nil {
		q.started <- struct{}{}
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (q *pipelineQueue) Ack(ctx context.Context, _ dispatchqueue.Delivery) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.acks++
	if q.ackFailOnce && q.acks == 1 {
		return errors.New("private SQS receipt")
	}
	return nil
}
func (q *pipelineQueue) Defer(ctx context.Context, _ dispatchqueue.Delivery) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.defers++
	return nil
}
func pipelineDelivery() dispatchqueue.Delivery {
	return dispatchqueue.Delivery{Body: `{"version":1,"batchId":"` + pipelineBatchID + `"}`, Receipt: "private-receipt"}
}
func pipelineFixture(t *testing.T) (*pipelineMemory, *pipelineQueue, *stubProvider, *DispatchPipeline) {
	t.Helper()
	vault := scheduledVault(t)
	repo := &scheduleMemory{}
	if _, err := NewScheduledSMSService(repo, vault).Enqueue(context.Background(), 7, []dtos.LegacySMS{futureSMS()}); err != nil {
		t.Fatal(err)
	}
	memory := &pipelineMemory{jobs: repo.jobs}
	queue := &pipelineQueue{}
	provider := &stubProvider{enabled: true}
	provider.response.MessageID = "550e8400-e29b-41d4-a716-446655440099"
	sender := NewScheduledSMSWorker(memory, provider, &countLimiter{allowed: true}, vault)
	return memory, queue, provider, NewDispatchPipeline(memory, queue, sender)
}
func TestDispatchConcurrentDuplicateReferencesSendOnce(t *testing.T) {
	repo, queue, provider, pipeline := pipelineFixture(t)
	var group sync.WaitGroup
	for range 32 {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := pipeline.consume(context.Background(), pipelineDelivery()); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	if len(provider.inputs) != 1 || repo.began != 1 || !repo.done || queue.acks == 0 {
		t.Fatal("duplicate reference replayed SMS")
	}
}
func TestDispatchUnknownAndLostAcknowledgmentNeverResend(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			_, queue, provider, pipeline := pipelineFixture(t)
			queue.ackFailOnce = true
			if unknown {
				provider.err = apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
			}
			if err := pipeline.consume(context.Background(), pipelineDelivery()); err == nil {
				t.Fatal("lost SQS acknowledgment hidden")
			}
			if err := pipeline.consume(context.Background(), pipelineDelivery()); err != nil {
				t.Fatal(err)
			}
			if len(provider.inputs) != 1 || queue.acks != 2 {
				t.Fatal("lost acknowledgment caused SMS replay")
			}
		})
	}
}
func TestDispatchPersistenceFailureKeepsReferenceUnacknowledged(t *testing.T) {
	repo, queue, provider, pipeline := pipelineFixture(t)
	repo.finishErr = errors.New("private SQL")
	if err := pipeline.consume(context.Background(), pipelineDelivery()); err == nil {
		t.Fatal("persistence failure hidden")
	}
	if err := pipeline.consume(context.Background(), pipelineDelivery()); err != nil {
		t.Fatal(err)
	}
	if queue.acks != 0 || queue.defers != 2 || len(provider.inputs) != 1 {
		t.Fatal("uncommitted dispatch acknowledged or replayed")
	}
}
func TestDispatchRateLimitDefersWithoutSubmitting(t *testing.T) {
	_, queue, provider, pipeline := pipelineFixture(t)
	budget := &countLimiter{allowed: false}
	pipeline.sender.limiter = budget
	if err := pipeline.consume(context.Background(), pipelineDelivery()); err != nil {
		t.Fatal(err)
	}
	if len(provider.inputs) != 0 || queue.acks != 0 || queue.defers != 1 {
		t.Fatal("blocked submission reached provider")
	}
	budget.allowed = true
	if err := pipeline.consume(context.Background(), pipelineDelivery()); err != nil {
		t.Fatal(err)
	}
	if len(provider.inputs) != 1 || queue.acks != 1 {
		t.Fatal("safe pre-submit deferral failed")
	}
}
func TestDispatchPublicationFailureRetainsOutboxIntent(t *testing.T) {
	repo, queue, provider, pipeline := pipelineFixture(t)
	repo.batch = &models.DispatchBatch{ID: pipelineBatchID, PublishAttempts: 1}
	queue.publishErr = errors.New("private AWS response")
	if err := pipeline.publish(context.Background()); err == nil {
		t.Fatal("publication error hidden")
	}
	queue.publishErr = nil
	if err := pipeline.publish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.publications) != 2 || repo.publications[0] || !repo.publications[1] || len(provider.inputs) != 0 {
		t.Fatal("outbox publication caused premature send")
	}
}
func TestDispatchMalformedReferenceDoesNotTouchSender(t *testing.T) {
	_, queue, provider, pipeline := pipelineFixture(t)
	message := pipelineDelivery()
	message.Body = `{"version":1,"message":"private-message"}`
	if err := pipeline.consume(context.Background(), message); err == nil {
		t.Fatal("poison reference accepted")
	}
	if queue.acks != 0 || queue.defers != 0 || len(provider.inputs) != 0 {
		t.Fatal("poison payload processed or discarded")
	}
}
func TestDispatchConsumersAreBoundedAndStopOnCancellation(t *testing.T) {
	_, queue, _, pipeline := pipelineFixture(t)
	queue.started = make(chan struct{}, workerConcurrency)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { pipeline.Run(ctx); close(done) }()
	deadline := time.After(5 * time.Second)
	for range workerConcurrency {
		select {
		case <-queue.started:
		case <-deadline:
			t.Fatal("consumers not started")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pipeline did not stop")
	}
	if queue.peak.Load() != workerConcurrency || queue.polling.Load() != 0 {
		t.Fatal("unbounded or leaked consumers")
	}
}

func TestDispatchRemainingWorkIsRepublishedBeforeAcknowledgment(t *testing.T) {
	for _, publishFails := range []bool{false, true} {
		t.Run(fmt.Sprint(publishFails), func(t *testing.T) {
			repo, queue, provider, pipeline := pipelineFixture(t)
			repo.moreWork = true
			if publishFails {
				queue.publishErr = errors.New("private AWS failure")
			}
			err := pipeline.consume(context.Background(), pipelineDelivery())
			if (err != nil) != publishFails {
				t.Fatal("handoff result inconsistent", err)
			}
			if len(provider.inputs) != 1 || queue.published != 1 {
				t.Fatal("remainder not handed off")
			}
			if publishFails && queue.acks != 0 {
				t.Fatal("old durable reference removed before replacement")
			}
			if !publishFails && queue.acks != 1 {
				t.Fatal("successful handoff not acknowledged")
			}
		})
	}
}

func TestDispatchReadyOutboxPublishesDespitePreparationFailure(t *testing.T) {
	repo, queue, provider, pipeline := pipelineFixture(t)
	repo.batch = &models.DispatchBatch{ID: pipelineBatchID, PublishAttempts: 1}
	repo.prepareErr = errors.New("new batch preparation unavailable")
	if err := pipeline.publish(context.Background()); err != nil {
		t.Fatal("ready outbox blocked by unrelated preparation failure", err)
	}
	if queue.published != 1 || len(repo.publications) != 1 || !repo.publications[0] || len(provider.inputs) != 0 {
		t.Fatal("ready reference not durably published")
	}
}
