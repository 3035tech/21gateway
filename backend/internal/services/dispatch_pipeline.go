package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/dispatchqueue"
	"github.com/21mobile/sms-gateway/internal/models"
)

const dispatchPublishTimeout = 10 * time.Second
const dispatchReceiveTimeout = 25 * time.Second
const dispatchWorkPause = 100 * time.Millisecond

type DispatchBatchRepository interface {
	Prepare(context.Context) error
	ClaimPublication(context.Context) (*models.DispatchBatch, error)
	PublicationResult(context.Context, models.DispatchBatch, bool, time.Duration) error
	ClaimBatch(context.Context, string) ([]models.ScheduledSMS, bool, error)
	Complete(context.Context, string) (bool, error)
}
type DispatchQueue interface {
	Publish(context.Context, string) error
	Receive(context.Context) ([]dispatchqueue.Delivery, error)
	Ack(context.Context, dispatchqueue.Delivery) error
	Defer(context.Context, dispatchqueue.Delivery) error
}
type DispatchPipeline struct {
	repo   DispatchBatchRepository
	queue  DispatchQueue
	sender *ScheduledSMSWorker
}

func NewDispatchPipeline(repo DispatchBatchRepository, queue DispatchQueue, sender *ScheduledSMSWorker) *DispatchPipeline {
	return &DispatchPipeline{repo: repo, queue: queue, sender: sender}
}
func (p *DispatchPipeline) Run(ctx context.Context) {
	var group sync.WaitGroup
	for range workerConcurrency {
		group.Add(2)
		go func() {
			defer group.Done()
			for ctx.Err() == nil {
				cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
				worked, err := p.publishNext(cycle)
				cancel()
				if err != nil && ctx.Err() == nil {
					slog.Error("Dispatch reference publication failed")
				}
				pause := workerTick
				if worked && err == nil {
					pause = dispatchWorkPause
				}
				if !dispatchPause(ctx, pause) {
					return
				}
			}
		}()
		go func() {
			defer group.Done()
			failures := 0
			for ctx.Err() == nil {
				poll, cancel := context.WithTimeout(ctx, dispatchReceiveTimeout)
				messages, err := p.queue.Receive(poll)
				cancel()
				if err != nil {
					if ctx.Err() == nil {
						slog.Error("Dispatch queue receive failed")
					}
					if failures < 8 {
						failures++
					}
					delay := CallbackBackoff(failures)
					if delay > 30*time.Second {
						delay = 30 * time.Second
					}
					if !dispatchPause(ctx, delay) {
						return
					}
					continue
				}
				failures = 0
				if len(messages) == 0 {
					if !dispatchPause(ctx, dispatchWorkPause) {
						return
					}
					continue
				}
				for _, message := range messages {
					cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
					err := p.consume(cycle, message)
					cancel()
					if err != nil && ctx.Err() == nil {
						slog.Error("Dispatch queue processing deferred")
						if !dispatchPause(ctx, workerTick) {
							return
						}
					}
				}
			}
		}()
	}
	group.Wait()
}
func (p *DispatchPipeline) publish(ctx context.Context) error {
	_, err := p.publishNext(ctx)
	return err
}
func (p *DispatchPipeline) publishNext(ctx context.Context) (bool, error) {
	// Drain durable publication work first. Preparing a new batch must not block
	// references already committed by an earlier cycle or another replica.
	batch, err := p.repo.ClaimPublication(ctx)
	if err != nil {
		return false, err
	}
	if batch == nil {
		if err := p.repo.Prepare(ctx); err != nil {
			return false, err
		}
		batch, err = p.repo.ClaimPublication(ctx)
		if err != nil || batch == nil {
			return false, err
		}
	}
	send, cancel := context.WithTimeout(ctx, dispatchPublishTimeout)
	publishErr := p.queue.Publish(send, batch.ID)
	cancel()
	finish, stop := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer stop()
	if err := p.repo.PublicationResult(finish, *batch, publishErr == nil, CallbackBackoff(batch.PublishAttempts)); err != nil {
		return true, err
	}
	// Retry only the reference publication, never a provider submission.
	return true, publishErr
}
func (p *DispatchPipeline) consume(ctx context.Context, message dispatchqueue.Delivery) error {
	id, err := dispatchqueue.ParseReference(message.Body)
	if err != nil {
		slog.Warn("Invalid dispatch reference received")
		// Keep the normal visibility timeout so poison payloads do not hot-loop.
		return err
	}
	jobs, done, err := p.repo.ClaimBatch(ctx, id)
	if err != nil {
		slog.Error("Dispatch batch claim failed", "batch_id", id)
		return p.deferFailure(ctx, message)
	}
	if done {
		return p.ackMessage(ctx, message)
	}
	if len(jobs) == 0 {
		return p.deferMessage(ctx, message)
	}
	progress, err := p.sender.processGroup(ctx, jobs)
	if err != nil {
		slog.Error("Dispatch submission outcome unavailable", "batch_id", id)
		return p.deferFailure(ctx, message)
	}
	done, err = p.repo.Complete(ctx, id)
	if err != nil {
		slog.Error("Dispatch batch completion failed", "batch_id", id)
		return p.deferFailure(ctx, message)
	}
	if !done {
		if !progress {
			return p.deferMessage(ctx, message)
		}
		// Hand off the remainder before acknowledging this delivery. If the publish
		// response is lost, the old reference remains available and DB guards dedupe.
		publish, cancel := context.WithTimeout(ctx, dispatchPublishTimeout)
		err := p.queue.Publish(publish, id)
		cancel()
		if err != nil {
			slog.Error("Dispatch remainder publication failed", "batch_id", id)
			return p.deferFailure(ctx, message)
		}
	}
	return p.ackMessage(ctx, message)
}
func (p *DispatchPipeline) ackMessage(ctx context.Context, message dispatchqueue.Delivery) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	return p.queue.Ack(finish, message)
}
func (p *DispatchPipeline) deferMessage(ctx context.Context, message dispatchqueue.Delivery) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	return p.queue.Defer(finish, message)
}
func dispatchPause(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (p *DispatchPipeline) deferFailure(ctx context.Context, message dispatchqueue.Delivery) error {
	_ = p.deferMessage(ctx, message)
	return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
}
