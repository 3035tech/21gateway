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
				if err != nil && ctx.Err() == nil {
					slog.Error("SMS reference publication failed")
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
			for ctx.Err() == nil {
				poll, cancel := context.WithTimeout(ctx, dispatchReceiveTimeout)
				messages, err := p.queue.Receive(poll)
				cancel()
				if err != nil {
					if ctx.Err() == nil {
						slog.Error("SMS reference receive failed")
					}
					if !dispatchPause(ctx, workerTick) {
						return
					}
					continue
				}
				for _, message := range messages {
					cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
					err := p.consume(cycle, message)
					cancel()
					if err != nil && ctx.Err() == nil {
						slog.Error("SMS reference processing deferred")
					}
				}
			}
		}()
	}
	group.Wait()
}
func (p *SendReferencePipeline) publishNext(ctx context.Context) (bool, error) {
	job, err := p.repo.ClaimPublication(ctx)
	if err != nil || job == nil {
		return false, err
	}
	send, cancel := context.WithTimeout(ctx, dispatchPublishTimeout)
	publishErr := p.queue.PublishSend(send, job.AttemptID)
	cancel()
	finish, stop := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer stop()
	if err := p.repo.PublicationResult(finish, *job, publishErr == nil, CallbackBackoff(1)); err != nil {
		return true, err
	}
	return true, publishErr
}
func (p *SendReferencePipeline) consume(ctx context.Context, message dispatchqueue.Delivery) error {
	id, err := dispatchqueue.ParseSendReference(message.Body)
	if err != nil {
		return err
	}
	job, done, ready, err := p.repo.Claim(ctx, id)
	if err != nil {
		return p.deferMessage(ctx, message)
	}
	if done {
		return p.ackMessage(ctx, message)
	}
	if !ready {
		return p.deferMessage(ctx, message)
	}
	_, err = p.sender.processGroup(ctx, []models.ScheduledSMS{job})
	if err != nil {
		return p.deferMessage(ctx, message)
	}
	// A pre-send release is durably queued for republication; acknowledge this
	// delivery so it cannot race the new reference.
	return p.ackMessage(ctx, message)
}
func (p *SendReferencePipeline) ackMessage(ctx context.Context, message dispatchqueue.Delivery) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	return p.queue.Ack(finish, message)
}
func (p *SendReferencePipeline) deferMessage(ctx context.Context, message dispatchqueue.Delivery) error {
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	return p.queue.Defer(finish, message)
}
