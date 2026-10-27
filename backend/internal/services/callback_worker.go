package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"
)

const workerConcurrency = 4
const workerTick = time.Second
const workerCycleTimeout = 20 * time.Second
const maxCallbackBackoff = 15 * time.Minute
const initialCallbackBackoff = 5 * time.Second
const callbackCleanupInterval = 5 * time.Minute

type CallbackSender interface {
	Deliver(context.Context, models.CallbackJob, models.SMSAttempt) (bool, string)
}
type CallbackWorker struct {
	repo   *repositories.Callbacks
	sender CallbackSender
}

func NewCallbackWorker(repo *repositories.Callbacks, sender CallbackSender) *CallbackWorker {
	return &CallbackWorker{repo: repo, sender: sender}
}
func CallbackBackoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 8 {
		attempts = 8
	}
	delay := initialCallbackBackoff * time.Duration(1<<uint(attempts-1))
	if delay > maxCallbackBackoff {
		delay = maxCallbackBackoff
	}
	return delay + time.Duration(rand.Int64N(int64(delay/4)+1))
}
func (w *CallbackWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(workerTick)
	defer ticker.Stop()
	cleanup := time.NewTicker(callbackCleanupInterval)
	defer cleanup.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanup.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, workerCycleTimeout)
			if err := w.repo.Cleanup(cleanupCtx); err != nil {
				slog.Error("Session cleanup failed")
			}
			cancel()
		case <-ticker.C:
			w.cycle(ctx)
		}
	}
}
func (w *CallbackWorker) cycle(ctx context.Context) {
	bounded, cancel := context.WithTimeout(ctx, workerCycleTimeout)
	defer cancel()
	if err := w.repo.Reconcile(bounded); err != nil {
		slog.Error("Callback reconciliation failed")
		return
	}
	var group sync.WaitGroup
	for i := 0; i < workerConcurrency; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := w.process(bounded); err != nil {
				slog.Error("Callback processing failed")
			}
		}()
	}
	group.Wait()
}
func (w *CallbackWorker) process(ctx context.Context) error {
	job, err := w.repo.Claim(ctx)
	if err != nil || job == nil {
		return err
	}
	if job.Attempts > repositories.CallbackMaxAttempts {
		err := w.repo.Finish(ctx, *job, false, models.CallbackNetworkError, 0)
		recordCallbackCompletion(ctx, models.CallbackKindDLR, job.ID, job.EventID, job.Attempts, false, models.CallbackNetworkError, err)
		return err
	}
	attempt, allowed, err := w.repo.DeliveryData(ctx, *job)
	if err != nil {
		return err
	}
	delivered := false
	code := models.CallbackAccessBlocked
	if allowed {
		delivered, code = w.sender.Deliver(ctx, *job, attempt)
	}
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	err = w.repo.Finish(finish, *job, delivered, code, CallbackBackoff(job.Attempts))
	recordCallbackCompletion(finish, models.CallbackKindDLR, job.ID, job.EventID, job.Attempts, delivered, code, err)
	return err
}
