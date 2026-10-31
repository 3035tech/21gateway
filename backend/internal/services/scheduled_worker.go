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
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
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
	repo          ScheduledWorkerRepository
	provider      SMSProvider
	limiter       SendLimiter
	vault         *security.ScheduledContentVault
	callbackLevel smsprovider.CallbackLevel
}

func NewScheduledSMSWorker(repo ScheduledWorkerRepository, provider SMSProvider, limiter SendLimiter, vault *security.ScheduledContentVault) *ScheduledSMSWorker {
	return &ScheduledSMSWorker{repo: repo, provider: provider, limiter: limiter, vault: vault, callbackLevel: smsprovider.CallbackNone}
}
func (w *ScheduledSMSWorker) SetCallbackLevel(level smsprovider.CallbackLevel) {
	w.callbackLevel = level
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
	logStage := func(stage string, err error) {
		slog.Warn("SMS processing deferred before provider request", "send_id", jobs[0].AttemptID, "credential_id", jobs[0].CredentialID, "stage", stage, "error", err)
	}
	release := func() (bool, error) {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
		defer cancel()
		err := w.repo.Release(finish, jobs)
		if err != nil {
			slog.Error("SMS release failed", "send_id", jobs[0].AttemptID, "error", err)
		}
		return false, err
	}
	reject := func(code apperrors.Code) (bool, error) {
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
		defer cancel()
		err := w.repo.Reject(finish, jobs, code)
		if err != nil {
			slog.Error("SMS rejection persistence failed", "send_id", jobs[0].AttemptID, "code", code, "error", err)
		} else {
			slog.Warn("SMS rejected before provider request", "send_id", jobs[0].AttemptID, "code", code)
		}
		return err == nil, err
	}
	credential, err := w.repo.Credential(ctx, jobs[0].CredentialID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return reject(apperrors.CodeForbidden)
	}
	if err != nil {
		logStage("credential_lookup", err)
		return release()
	}
	if !credential.ValidAt(time.Now()) || (credential.CompanyID != nil && (credential.CompanyActive == nil || !*credential.CompanyActive)) {
		return reject(apperrors.CodeForbidden)
	}
	credential.PasswordHash = ""
	messages := make([]dtos.LegacySMS, len(jobs))
	for i, job := range jobs {
		if job.CredentialID != credential.ID || job.Attempt.ProviderCode != jobs[0].Attempt.ProviderCode {
			return reject(apperrors.CodeConflict)
		}
		text, err := w.vault.Decrypt(job.CredentialID, job.EventKey, job.EncryptedMessage)
		if err != nil {
			logStage("message_decryption", err)
			return release()
		}
		messages[i] = dtos.LegacySMS{Destination: strings.TrimPrefix(job.Attempt.Destination, brazilCountryCode), MessageText: text, CorrelationID: job.Attempt.CorrelationID}
	}
	credential.CompanySMSProvider = jobs[0].Attempt.ProviderCode
	provider := w.provider
	if scoped, ok := provider.(interface {
		Prepare(context.Context) (SMSProvider, error)
	}); ok {
		resolved, err := scoped.Prepare(WithCredential(ctx, credential))
		if err != nil {
			logStage("provider_configuration", err)
			return release()
		}
		provider = resolved
	}
	if !provider.Enabled() {
		logStage("provider_disabled", nil)
		return release()
	}
	budget := w.limiter
	if partial, ok := budget.(interface {
		ReserveUpTo(context.Context, uint64, int) (int, error)
	}); ok {
		granted, err := partial.ReserveUpTo(ctx, credential.ID, len(jobs))
		if err != nil || granted <= 0 {
			logStage("send_budget", err)
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
	service.SetCallbackLevel(w.callbackLevel)
	result, sendErr := service.Send(WithCredential(ctx, credential), credential.ID, messages)
	if adapter.started {
		return adapter.finishErr == nil, adapter.finishErr
	}
	// Before Begin there was no provider request: deferral is safe, not an SMS retry.
	if sendErr != nil {
		logStage("begin_submission", sendErr)
		return release()
	}
	if len(result) > 0 && result[0].Code == dtos.LegacyCodeRateLimited {
		logStage("rate_limited", nil)
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
