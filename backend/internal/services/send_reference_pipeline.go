package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/21mobile/sms-gateway/internal/integrations/dispatchqueue"
	"github.com/21mobile/sms-gateway/internal/models"
)

type SendReferenceRepository interface {
	ClaimPublication(context.Context) (*models.ScheduledSMS, error)
	PublicationResult(context.Context, models.ScheduledSMS, bool, time.Duration) error
	Claim(context.Context, uint64) (models.ScheduledSMS, bool, bool, error)
}
type SendReferenceQueue interface {
	PublishSend(context.Context, uint64) error
	Receive(context.Context) ([]dispatchqueue.Delivery, error)
	Ack(context.Context, dispatchqueue.Delivery) error
	Defer(context.Context, dispatchqueue.Delivery) error
}
type SendReferencePipeline struct {
	repo   SendReferenceRepository
	queue  SendReferenceQueue
	sender *ScheduledSMSWorker
}

func NewSendReferencePipeline(repo SendReferenceRepository, queue SendReferenceQueue, sender *ScheduledSMSWorker) *SendReferencePipeline {
	return &SendReferencePipeline{repo: repo, queue: queue, sender: sender}
}
func (p *SendReferencePipeline) Run(ctx context.Context) {
	var group sync.WaitGroup
	for range workerConcurrency {
		group.Add(2)
		go func() {
			defer group.Done()
			for ctx.Err() == nil {
				cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
				worked, err := p.publishNext(cycle)
				cancel()
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
						slog.Error("SMS reference receive failed")
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
				for _, message := range messages {
					cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
					_ = p.consume(cycle, message)
					cancel()
				}
			}
		}()
	}
	group.Wait()
}
func (p *SendReferencePipeline) publishNext(ctx context.Context) (bool, error) {
	job, err := p.repo.ClaimPublication(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Error("SMS reference claim failed")
		}
		return false, err
	}
	if job == nil {
		return false, err
	}
	send, cancel := context.WithTimeout(ctx, dispatchPublishTimeout)
	publishErr := p.queue.PublishSend(send, job.AttemptID)
	cancel()
	finish, stop := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer stop()
	if err := p.repo.PublicationResult(finish, *job, publishErr == nil, CallbackBackoff(job.PublishAttempts)); err != nil {
		slog.Error("SMS reference publication state failed", "send_id", job.AttemptID, "publish_attempts", job.PublishAttempts)
		return true, err
	}
	if publishErr != nil {
		slog.Warn("SMS reference publish failed", "send_id", job.AttemptID, "publish_attempts", job.PublishAttempts)
	} else {
		slog.Info("SMS reference published", "send_id", job.AttemptID, "publish_attempts", job.PublishAttempts)
	}
	return true, publishErr
}
func (p *SendReferencePipeline) consume(ctx context.Context, message dispatchqueue.Delivery) error {
	id, err := dispatchqueue.ParseSendReference(message.Body)
	if err != nil {
		slog.Warn("SMS reference payload invalid")
		return err
	}
	job, done, ready, err := p.repo.Claim(ctx, id)
	if err != nil {
		slog.Warn("SMS reference claim deferred", "send_id", id)
		return p.deferMessage(ctx, message, id)
	}
	if done {
		slog.Info("SMS reference already complete", "send_id", id)
		return p.ackMessage(ctx, message, id)
	}
	if !ready {
		slog.Debug("SMS reference not ready", "send_id", id)
		return p.deferMessage(ctx, message, id)
	}
	progress, err := p.sender.processGroup(ctx, []models.ScheduledSMS{job})
	if err != nil {
		slog.Warn("SMS reference processing deferred", "send_id", id)
		return p.deferMessage(ctx, message, id)
	}
	if progress {
		slog.Info("SMS reference processing persisted", "send_id", id)
	} else {
		slog.Info("SMS reference released for retry", "send_id", id)
	}
	// A pre-send release is durably queued for republication; acknowledge this
	// delivery so it cannot race the new reference.
	return p.ackMessage(ctx, message, id)
}
func (p *SendReferencePipeline) ackMessage(ctx context.Context, message dispatchqueue.Delivery, id uint64) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	if err := p.queue.Ack(finish, message); err != nil {
		slog.Warn("SMS reference acknowledgment failed", "send_id", id)
		return err
	}
	return nil
}
func (p *SendReferencePipeline) deferMessage(ctx context.Context, message dispatchqueue.Delivery, id uint64) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	if err := p.queue.Defer(finish, message); err != nil {
		slog.Warn("SMS reference visibility update failed", "send_id", id)
		return err
	}
	return nil
}
