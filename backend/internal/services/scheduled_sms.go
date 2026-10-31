package services

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
)

const maximumScheduleTimestamp int64 = 253402300799

type ScheduleAcceptor interface {
	Enqueue(context.Context, uint64, []dtos.LegacySMS) ([]dtos.LegacySendResult, error)
}
type ScheduledQueueWriter interface {
	Enqueue(context.Context, []models.ScheduledSMS) ([]models.ScheduledSMS, error)
}
type ScheduledSMSService struct {
	repo    ScheduledQueueWriter
	vault   *security.ScheduledContentVault
	history *security.SMSHistoryVault
}

func NewScheduledSMSService(repo ScheduledQueueWriter, vault *security.ScheduledContentVault) *ScheduledSMSService {
	return &ScheduledSMSService{repo: repo, vault: vault}
}
func (s *ScheduledSMSService) SetHistoryVault(vault *security.SMSHistoryVault) { s.history = vault }
func (s *ScheduledSMSService) Enqueue(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS) ([]dtos.LegacySendResult, error) {
	return s.enqueue(ctx, credentialID, messages, false)
}

// EnqueueAll persists immediate and future messages before any provider call.
func (s *ScheduledSMSService) EnqueueAll(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS) ([]dtos.LegacySendResult, error) {
	return s.enqueue(ctx, credentialID, messages, true)
}
func (s *ScheduledSMSService) enqueue(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS, allowImmediate bool) ([]dtos.LegacySendResult, error) {
	if !s.vault.Enabled() {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if len(messages) == 0 || len(messages) > models.ScheduledBatchSize {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	result := make([]dtos.LegacySendResult, len(messages))
	jobs := make([]models.ScheduledSMS, 0, len(messages))
	positions := make([]int, 0, len(messages))
	now := time.Now().UTC()
	for i, input := range messages {
		code := apperrors.CodeNone
		reason := ""
		if input.ScheduledDate != nil && *input.ScheduledDate != 0 {
			if *input.ScheduledDate <= now.Unix() || *input.ScheduledDate > maximumScheduleTimestamp {
				code = apperrors.CodeInvalidRequest
				reason = "scheduled_date_out_of_range"
			}
		} else if !allowImmediate {
			code = apperrors.CodeInvalidRequest
			reason = "scheduled_date_required"
		}
		normalized, normalizationCode := prepareSMSText(input.MessageText)
		plain := input
		plain.MessageText = normalized
		plain.ScheduledDate = nil
		if normalizationCode != apperrors.CodeNone {
			code = normalizationCode
			reason = "message_normalization"
		} else if validation := validateMessage(plain); validation != apperrors.CodeNone {
			code = validation
			if validation == apperrors.CodeInvalidRequest {
				reason = "correlation_id_too_long"
			} else {
				reason = string(validation)
			}
		}
		if code != apperrors.CodeNone {
			slog.Warn("SMS item rejected before enqueue", "credential_id", credentialID, "item_index", i, "code", code, "reason", reason)
			result[i] = failure(input, code)
			continue
		}
		key := uuid.NewString()
		ciphertext, err := s.vault.Encrypt(credentialID, key, normalized)
		if err != nil {
			return nil, err
		}
		due := now
		var scheduledAt *time.Time
		if input.ScheduledDate != nil && *input.ScheduledDate != 0 {
			due = time.Unix(*input.ScheduledDate, 0).UTC()
			scheduledAt = &due
		}
		attempt := models.SMSAttempt{CredentialID: credentialID, Destination: brazilCountryCode + input.Destination, CorrelationID: input.CorrelationID, Encoding: whitelabel.SelectEncoding(normalized), State: models.SubmissionPending, ScheduledAt: scheduledAt}
		if credential, ok := CredentialFromContext(ctx); ok && credential.ID == credentialID {
			attempt.CompanyID = credential.CompanyID
			attempt.ProviderCode = selectedSMSProvider(credential.CompanySMSProvider)
		}
		if s.history != nil {
			reference := uuid.NewString()
			historyCiphertext, err := s.history.Encrypt(credentialID, reference, normalized)
			if err != nil {
				return nil, err
			}
			attempt.ContentRef = &reference
			attempt.EncryptedMessage = historyCiphertext
		}
		jobs = append(jobs, models.ScheduledSMS{CredentialID: credentialID, EventKey: key, EncryptedMessage: ciphertext, DueAt: due, AvailableAt: due, State: models.ScheduleQueued, SendReference: allowImmediate, Attempt: attempt})
		positions = append(positions, i)
	}
	if len(jobs) == 0 {
		return result, nil
	}
	saved, err := s.repo.Enqueue(ctx, jobs)
	if err != nil {
		slog.Error("SMS enqueue failed", "credential_id", credentialID, "items", len(jobs), "error", dbError(err))
		return nil, dbError(err)
	}
	if len(saved) != len(jobs) {
		slog.Error("SMS enqueue result incomplete", "credential_id", credentialID, "expected", len(jobs), "actual", len(saved))
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	for i, job := range saved {
		position := positions[i]
		result[position] = dtos.LegacySendResult{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, IDSMS: strconv.FormatUint(job.AttemptID, 10), CorrelationID: messages[position].CorrelationID}
		if allowImmediate {
			slog.Info("SMS queued for SQS", "send_id", job.AttemptID, "scheduled", job.Attempt.ScheduledAt != nil)
		}
	}
	return result, nil
}

// Called only during startup. Each HTTP request reserves the shared budget once.
func (s *SMSService) SetScheduler(queue ScheduleAcceptor) { s.scheduler = queue }
func (s *SMSService) SetDispatchQueue(queue interface {
	EnqueueAll(context.Context, uint64, []dtos.LegacySMS) ([]dtos.LegacySendResult, error)
}) {
	s.asyncQueue = queue
}

type reservedSendBudget struct{}

func (reservedSendBudget) Allow(context.Context, uint64, int) (bool, error) { return true, nil }
func (s *SMSService) sendWithSchedules(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS) ([]dtos.LegacySendResult, error) {
	immediate, future := []dtos.LegacySMS{}, []dtos.LegacySMS{}
	immediatePositions, futurePositions := []int{}, []int{}
	for i, m := range messages {
		if m.ScheduledDate != nil && *m.ScheduledDate != 0 {
			future = append(future, m)
			futurePositions = append(futurePositions, i)
		} else {
			immediate = append(immediate, m)
			immediatePositions = append(immediatePositions, i)
		}
	}
	allowed, err := s.allow(ctx, credentialID, len(immediate))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !allowed {
		results := make([]dtos.LegacySendResult, len(messages))
		for i, m := range messages {
			results[i] = failure(m, apperrors.CodeRateLimited)
		}
		return results, nil
	}
	// Queue before sending any immediate item. A queue failure cannot follow a live send.
	queued, err := s.scheduler.Enqueue(ctx, credentialID, future)
	if err != nil {
		return nil, err
	}
	results := make([]dtos.LegacySendResult, len(messages))
	for i, position := range futurePositions {
		results[position] = queued[i]
	}
	if len(immediate) > 0 {
		service := NewSMSServiceWithLimiter(s.repository, s.provider, reservedSendBudget{})
		sent, err := service.Send(ctx, credentialID, immediate)
		if err != nil {
			for i, m := range immediate {
				results[immediatePositions[i]] = failure(m, apperrors.CodeServiceUnavailable)
			}
		} else {
			for i, position := range immediatePositions {
				results[position] = sent[i]
			}
		}
	}
	return results, nil
}
