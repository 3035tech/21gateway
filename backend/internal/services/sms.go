package services

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/time/rate"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
)

const (
	MaximumMessageCharacters     = 160
	MaximumCorrelationCharacters = 50
	completionTimeout            = 3 * time.Second
	brazilCountryCode            = "55"
	sendRateWindow               = time.Second
	sendRateBurst                = 1
)

var brazilLocalNumber = regexp.MustCompile(`^[0-9]{11}$`)

type SMSProvider interface {
	Enabled() bool
	Send(context.Context, whitelabel.SendRequest) (whitelabel.SendResponse, error)
}
type AttemptRepository interface {
	Create(context.Context, *models.SMSAttempt) error
	Complete(context.Context, models.SMSAttempt) error
}

type SendLimiter interface {
	Allow(context.Context, uint64, int) (bool, error)
}

type SMSService struct {
	scheduler  ScheduleAcceptor
	limiter    SendLimiter
	repository AttemptRepository
	provider   SMSProvider
	limits     sync.Map
}

func NewSMSService(repository AttemptRepository, provider SMSProvider) *SMSService {
	return &SMSService{repository: repository, provider: provider}
}

func (s *SMSService) Send(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS) ([]dtos.LegacySendResult, error) {
	if len(messages) < 1 || len(messages) > whitelabel.MaximumBatchMessages {
		return []dtos.LegacySendResult{failure(dtos.LegacySMS{}, apperrors.CodeInvalidRequest)}, nil
	}
	if s.scheduler != nil {
		for _, m := range messages {
			if m.ScheduledDate != nil && *m.ScheduledDate != 0 {
				return s.sendWithSchedules(ctx, credentialID, messages)
			}
		}
	}
	if len(messages) > 1 {
		return s.sendBatch(ctx, credentialID, messages)
	}
	input := messages[0]
	normalized, normalizationCode := prepareSMSText(input.MessageText)
	if normalizationCode != apperrors.CodeNone {
		return []dtos.LegacySendResult{failure(input, normalizationCode)}, nil
	}
	input.MessageText = normalized
	if code := validateMessage(input); code != apperrors.CodeNone {
		return []dtos.LegacySendResult{failure(input, code)}, nil
	}
	provider := s.provider
	if scoped, ok := provider.(interface {
		Prepare(context.Context) (SMSProvider, error)
	}); ok {
		var err error
		provider, err = scoped.Prepare(ctx)
		if err != nil {
			return nil, err
		}
	}
	if !provider.Enabled() {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	allowed, err := s.allow(ctx, credentialID, 1)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !allowed {
		return []dtos.LegacySendResult{failure(input, apperrors.CodeRateLimited)}, nil
	}
	encoding := whitelabel.SelectEncoding(input.MessageText)
	attempt := models.SMSAttempt{CredentialID: credentialID, Destination: brazilCountryCode + input.Destination, CorrelationID: input.CorrelationID, Encoding: encoding, State: models.SubmissionPending}
	if credential, ok := CredentialFromContext(ctx); ok && credential.ID == credentialID {
		attempt.CompanyID = credential.CompanyID
	}
	if err := s.repository.Create(ctx, &attempt); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	id := strconv.FormatUint(attempt.ID, 10)
	sent, err := provider.Send(ctx, whitelabel.SendRequest{Type: whitelabel.MessageTypeSMS, Message: input.MessageText, Encoding: encoding, Destination: attempt.Destination, ReferenceID: input.CorrelationID, Index: id, CallbackLevel: whitelabel.CallbackNone})
	if err != nil {
		code := apperrors.CodeUnknownOutcome
		var typed *apperrors.Error
		if errors.As(err, &typed) {
			code = typed.Code
		}
		attempt.State = models.SubmissionRejected
		if code == apperrors.CodeUnknownOutcome {
			attempt.State = models.SubmissionUnknown
		}
		attempt.FailureCode = code
		s.complete(ctx, attempt)
		result := failure(input, code)
		if attempt.State == models.SubmissionUnknown {
			result.IDSMS = id
		}
		return []dtos.LegacySendResult{result}, nil
	}
	attempt.State = models.SubmissionAccepted
	attempt.ProviderMessageID = &sent.MessageID
	s.complete(ctx, attempt)
	return []dtos.LegacySendResult{{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, IDSMS: id, CorrelationID: input.CorrelationID}}, nil
}

func (s *SMSService) complete(ctx context.Context, attempt models.SMSAttempt) {
	// Persist the final outcome even if the caller disconnected. Never resend.
	completion, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	if err := s.repository.Complete(completion, attempt); err != nil {
		// An accepted SMS remains accepted to the caller, even if the local update failed.
		slog.Error("SMS outcome persistence failed", "attempt_id", attempt.ID, "state", attempt.State, "provider_message_id", attempt.ProviderMessageID)
	}
}

func validateMessage(input dtos.LegacySMS) apperrors.Code {
	if !brazilLocalNumber.MatchString(input.Destination) {
		return apperrors.CodeInvalidDestination
	}
	if !utf8.ValidString(input.MessageText) || strings.TrimSpace(input.MessageText) == "" || utf8.RuneCountInString(input.MessageText) > MaximumMessageCharacters {
		return apperrors.CodeInvalidMessage
	}
	if utf8.RuneCountInString(input.CorrelationID) > MaximumCorrelationCharacters {
		return apperrors.CodeInvalidRequest
	}
	if input.ScheduledDate != nil && *input.ScheduledDate != 0 {
		return apperrors.CodeScheduleUnsupported
	}
	return apperrors.CodeNone
}

func failure(input dtos.LegacySMS, code apperrors.Code) dtos.LegacySendResult {
	result := dtos.LegacySendResult{Status: dtos.LegacyStatusNOK, Code: dtos.LegacyCodeFailed, CorrelationID: input.CorrelationID}
	switch code {
	case apperrors.CodeInvalidDestination:
		result.Message = dtos.LegacyInvalidNumber
	case apperrors.CodeInvalidMessage:
		result.Message = dtos.LegacyInvalidMessage
	case apperrors.CodeInvalidRequest:
		result.Message = dtos.LegacyInvalidRequest
	case apperrors.CodeBatchUnsupported:
		result.Message = dtos.LegacyBatchUnsupported
	case apperrors.CodeScheduleUnsupported:
		result.Message = dtos.LegacyScheduleUnsupported
	case apperrors.CodeProviderUnauthorized:
		result.Message = dtos.LegacyProviderUnauthorized
	case apperrors.CodeInsufficientBalance:
		result.Message = dtos.LegacyInsufficientBalance
	case apperrors.CodeRateLimited:
		result.Message = dtos.LegacyRateLimited
		result.Code = dtos.LegacyCodeRateLimited
	case apperrors.CodeProviderRejected:
		result.Message = dtos.LegacyProviderRejected
	case apperrors.CodeUnknownOutcome:
		result.Message = dtos.LegacyUnknownOutcome
	default:
		result.Message = dtos.LegacyGeneralError
	}
	return result
}

// Limits apply per credential in this process. Multi-replica deployments require a shared limiter.
func (s *SMSService) allowSend(credentialID uint64) bool {
	cached, ok := s.limits.Load(credentialID)
	if !ok {
		cached, _ = s.limits.LoadOrStore(credentialID, rate.NewLimiter(rate.Every(sendRateWindow), sendRateBurst))
	}
	return cached.(*rate.Limiter).Allow()
}

func NewSMSServiceWithLimiter(repo AttemptRepository, provider SMSProvider, limiter SendLimiter) *SMSService {
	service := NewSMSService(repo, provider)
	service.limiter = limiter
	return service
}
func (s *SMSService) allow(ctx context.Context, id uint64, count int) (bool, error) {
	if s.limiter != nil {
		return s.limiter.Allow(ctx, id, count)
	}
	return s.allowSend(id), nil
}
