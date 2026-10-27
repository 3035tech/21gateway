package services

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type ScheduledWorkerRepository interface {
	Claim(context.Context) ([]models.ScheduledSMS, error)
	Credential(context.Context, uint64) (models.APICredential, error)
	Begin(context.Context, []models.ScheduledSMS) error
	Release(context.Context, []models.ScheduledSMS) error
	Reject(context.Context, []models.ScheduledSMS, apperrors.Code) error
	Finish(context.Context, []models.ScheduledSMS, []models.SMSAttempt) error
}
type ScheduledSMSWorker struct {
	repo     ScheduledWorkerRepository
	provider SMSProvider
	limiter  SendLimiter
	vault    *security.ScheduledContentVault
}

func NewScheduledSMSWorker(repo ScheduledWorkerRepository, provider SMSProvider, limiter SendLimiter, vault *security.ScheduledContentVault) *ScheduledSMSWorker {
	return &ScheduledSMSWorker{repo: repo, provider: provider, limiter: limiter, vault: vault}
}
func (w *ScheduledSMSWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(workerTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cycle, cancel := context.WithTimeout(ctx, workerCycleTimeout)
			var group sync.WaitGroup
			for range workerConcurrency {
				group.Add(1)
				go func() {
					defer group.Done()
					if err := w.process(cycle); err != nil {
						slog.Error("Scheduled SMS processing failed")
					}
				}()
			}
			group.Wait()
			cancel()
		}
	}
}
func (w *ScheduledSMSWorker) process(ctx context.Context) error {
	jobs, err := w.repo.Claim(ctx)
	if err != nil || len(jobs) == 0 {
		return err
	}
	return w.processJobs(ctx, jobs)
}
func (w *ScheduledSMSWorker) processJobs(ctx context.Context, jobs []models.ScheduledSMS) error {
	_, err := w.processGroup(ctx, jobs)
	return err
}
func (w *ScheduledSMSWorker) processGroup(ctx context.Context, jobs []models.ScheduledSMS) (bool, error) {
	if len(jobs) == 0 {
		return false, nil
	}
	release := func() (bool, error) {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
		defer cancel()
		return false, w.repo.Release(finish, jobs)
	}
	reject := func(code apperrors.Code) (bool, error) {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
		defer cancel()
		err := w.repo.Reject(finish, jobs, code)
		return err == nil, err
	}
	credential, err := w.repo.Credential(ctx, jobs[0].CredentialID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return reject(apperrors.CodeForbidden)
	}
	if err != nil {
		return release()
	}
	if !credential.ValidAt(time.Now()) || (credential.CompanyID != nil && (credential.CompanyActive == nil || !*credential.CompanyActive)) {
		return reject(apperrors.CodeForbidden)
	}
	credential.PasswordHash = ""
	messages := make([]dtos.LegacySMS, len(jobs))
	for i, job := range jobs {
		if job.CredentialID != credential.ID {
			return reject(apperrors.CodeConflict)
		}
		text, err := w.vault.Decrypt(job.CredentialID, job.EventKey, job.EncryptedMessage)
		if err != nil {
			return release()
		}
		messages[i] = dtos.LegacySMS{Destination: strings.TrimPrefix(job.Attempt.Destination, brazilCountryCode), MessageText: text, CorrelationID: job.Attempt.CorrelationID}
	}
	provider := w.provider
	if scoped, ok := provider.(interface {
		Prepare(context.Context) (SMSProvider, error)
	}); ok {
		resolved, err := scoped.Prepare(WithCredential(ctx, credential))
		if err != nil {
			return release()
		}
		provider = resolved
	}
	if !provider.Enabled() {
		return release()
	}
	budget := w.limiter
	if partial, ok := budget.(interface {
		ReserveUpTo(context.Context, uint64, int) (int, error)
	}); ok {
		granted, err := partial.ReserveUpTo(ctx, credential.ID, len(jobs))
		if err != nil || granted <= 0 {
			return release()
		}
		if granted > len(jobs) {
			return reject(apperrors.CodeConflict)
		}
		if granted < len(jobs) {
			finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
			err := w.repo.Release(finish, jobs[granted:])
			cancel()
			if err != nil {
				return false, err
			}
			jobs = jobs[:granted]
			messages = messages[:granted]
		}
		budget = reservedSendBudget{}
	}
	adapter := &scheduledAttempts{repo: w.repo, jobs: jobs}
	service := NewSMSServiceWithLimiter(adapter, provider, budget)
	result, sendErr := service.Send(WithCredential(ctx, credential), credential.ID, messages)
	if adapter.started {
		return adapter.finishErr == nil, adapter.finishErr
	}
	// Before Begin there was no provider request: deferral is safe, not an SMS retry.
	if sendErr != nil {
		return release()
	}
	if len(result) > 0 && result[0].Code == dtos.LegacyCodeRateLimited {
		return release()
	}
	return reject(apperrors.CodeInvalidRequest)
}

// Reuse the immediate sender while preserving the IDs accepted at schedule time.
// Begin is committed before any network call. Once begun, an expired lease is unknown.
type scheduledAttempts struct {
	repo      ScheduledWorkerRepository
	jobs      []models.ScheduledSMS
	started   bool
	finishErr error
}

func (a *scheduledAttempts) Create(ctx context.Context, attempt *models.SMSAttempt) error {
	return a.CreateBatch(ctx, []*models.SMSAttempt{attempt})
}
func (a *scheduledAttempts) CreateBatch(ctx context.Context, attempts []*models.SMSAttempt) error {
	if len(attempts) != len(a.jobs) {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	for i, attempt := range attempts {
		if attempt.CredentialID != a.jobs[i].CredentialID {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		attempt.ID = a.jobs[i].AttemptID
	}
	if err := a.repo.Begin(ctx, a.jobs); err != nil {
		return err
	}
	a.started = true
	return nil
}
func (a *scheduledAttempts) Complete(ctx context.Context, attempt models.SMSAttempt) error {
	return a.CompleteBatch(ctx, []models.SMSAttempt{attempt})
}
func (a *scheduledAttempts) CompleteBatch(ctx context.Context, attempts []models.SMSAttempt) error {
	a.finishErr = a.repo.Finish(ctx, a.jobs, attempts)
	return a.finishErr
}
