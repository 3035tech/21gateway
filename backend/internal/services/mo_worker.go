package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"log/slog"
	"sync"
	"time"
)

type MOWorkerRepository interface {
	Reconcile(context.Context) error
	Claim(context.Context) (*models.MOCallbackJob, error)
	DeliveryData(context.Context, models.MOCallbackJob) (models.MOEvent, models.SMSAttempt, bool, error)
	Finish(context.Context, models.MOCallbackJob, bool, string, time.Duration) error
}
type MOSender interface {
	DeliverMO(context.Context, string, models.MOMessage) (bool, string)
}
type MOWorker struct {
	repo   MOWorkerRepository
	sender MOSender
	vault  *security.MOContentVault
}

func NewMOWorker(repo MOWorkerRepository, sender MOSender, vault *security.MOContentVault) *MOWorker {
	return &MOWorker{repo: repo, sender: sender, vault: vault}
}
func (w *MOWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(workerTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
			if err := w.repo.Reconcile(cycle); err != nil {
				slog.Error("MO reconciliation failed")
				cancel()
				continue
			}
			var group sync.WaitGroup
			for range workerConcurrency {
				group.Add(1)
				go func() {
					defer group.Done()
					if err := w.process(cycle); err != nil {
						slog.Error("MO processing failed")
					}
				}()
			}
			group.Wait()
			cancel()
		}
	}
}
func (w *MOWorker) process(ctx context.Context) error {
	job, err := w.repo.Claim(ctx)
	if err != nil || job == nil {
		return err
	}
	if job.Attempts > repositories.CallbackMaxAttempts {
		err := w.repo.Finish(ctx, *job, false, models.CallbackNetworkError, 0)
		recordCallbackCompletion(ctx, models.CallbackKindMO, job.ID, job.EventID, job.Attempts, false, models.CallbackNetworkError, err)
		return err
	}
	event, attempt, allowed, err := w.repo.DeliveryData(ctx, *job)
	if err != nil {
		return err
	}
	delivered := false
	code := models.CallbackAccessBlocked
	if allowed {
		text, err := w.vault.Decrypt(event.CredentialID, event.EventKey, event.EncryptedMessage)
		if err != nil {
			code = models.CallbackContentUnavailable
		} else {
			message := models.MOMessage{ID: event.ID, AttemptID: attempt.ID, CorrelationID: attempt.CorrelationID, PhoneNumber: event.PhoneNumber, Message: text, ReceivedAt: event.ReceivedAt}
			delivered, code = w.sender.DeliverMO(ctx, job.URL, message)
		}
	}
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	err = w.repo.Finish(finish, *job, delivered, code, CallbackBackoff(job.Attempts))
	recordCallbackCompletion(finish, models.CallbackKindMO, job.ID, job.EventID, job.Attempts, delivered, code, err)
	return err
}
