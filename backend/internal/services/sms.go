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
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
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
	asyncQueue interface {
		EnqueueAll(context.Context, uint64, []dtos.LegacySMS) ([]dtos.LegacySendResult, error)
	}
	limiter       SendLimiter
	repository    AttemptRepository
	provider      SMSProvider
	limits        sync.Map
	history       *security.SMSHistoryVault
	callbackLevel whitelabel.CallbackLevel
}

func NewSMSService(repository AttemptRepository, provider SMSProvider) *SMSService {
	return &SMSService{repository: repository, provider: provider, callbackLevel: whitelabel.CallbackNone}
}
func (s *SMSService) SetHistoryVault(vault *security.SMSHistoryVault) { s.history = vault }
func (s *SMSService) SetCallbackLevel(level whitelabel.CallbackLevel) { s.callbackLevel = level }
func (s *SMSService) Queued() bool                                    { return s.asyncQueue != nil }

// A company token test calls Zoug directly so the response verifies the saved token.
func (s *SMSService) SendCompanyTest(ctx context.Context, company models.Company, destination, message, correlationID string) (string, error) {
	preparer, ok := s.provider.(interface {
		PrepareCompany(models.Company) (SMSProvider, error)
	})
	if !ok {
		return "", apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	provider, err := preparer.PrepareCompany(company)
	if err != nil {
		return "", err
	}
	if !s.allowCompanyTest(company.ID) {
		return "", apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	normalized, code := prepareSMSText(message)
	if code != apperrors.CodeNone {
		return "", apperrors.Wrap(code, nil)
	}
	response, err := provider.Send(ctx, whitelabel.SendRequest{Type: whitelabel.MessageTypeSMS, Message: normalized, Encoding: whitelabel.SelectEncoding(normalized), Destination: brazilCountryCode + destination, ReferenceID: correlationID, Index: strconv.FormatUint(company.ID, 10), CallbackLevel: whitelabel.CallbackNone})
	if err != nil {
		return response.MessageID, err
	}
	return response.MessageID, nil
}

func (s *SMSService) DiagnoseCompany(ctx context.Context, company models.Company, runID string, actorID uint64) ([]whitelabel.DiagnosticResult, error) {
	preparer, ok := s.provider.(interface {
		PrepareCompany(models.Company) (SMSProvider, error)
	})
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	provider, err := preparer.PrepareCompany(company)
	if err != nil {
		return nil, err
	}
	client, ok := provider.(*whitelabel.Client)
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	return client.Diagnose(ctx, runID, company.ID, actorID), nil
}

func (s *SMSService) allowCompanyTest(companyID uint64) bool {
	key := "company-test:" + strconv.FormatUint(companyID, 10)
	cached, _ := s.limits.LoadOrStore(key, rate.NewLimiter(rate.Every(sendRateWindow), sendRateBurst))
	return cached.(*rate.Limiter).Allow()
}

func (s *SMSService) allowCompanyDiagnostic(companyID uint64) bool {
	key := "company-diagnostic:" + strconv.FormatUint(companyID, 10)
	cached, _ := s.limits.LoadOrStore(key, rate.NewLimiter(rate.Every(time.Minute), 1))
	return cached.(*rate.Limiter).Allow()
}

func (s *SMSService) encryptAttempt(attempt *models.SMSAttempt, message string) error {
	if s.history == nil {
		return nil
	}
	reference := uuid.NewString()
	encrypted, err := s.history.Encrypt(attempt.CredentialID, reference, message)
	if err != nil {
		return err
	}
	attempt.ContentRef = &reference
	attempt.EncryptedMessage = encrypted
	return nil
}

func (s *SMSService) Send(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS) ([]dtos.LegacySendResult, error) {
	if len(messages) < 1 || len(messages) > whitelabel.MaximumBatchMessages {
		return []dtos.LegacySendResult{failure(dtos.LegacySMS{}, apperrors.CodeInvalidRequest)}, nil
	}
	if s.asyncQueue != nil {
		allowed, err := s.allow(ctx, credentialID, 0)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
		}
		if !allowed {
			results := make([]dtos.LegacySendResult, len(messages))
			for i, input := range messages {
				results[i] = failure(input, apperrors.CodeRateLimited)
			}
			return results, nil
		}
		return s.asyncQueue.EnqueueAll(ctx, credentialID, messages)
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
	if err := s.encryptAttempt(&attempt, input.MessageText); err != nil {
		return nil, err
	}
	if err := s.repository.Create(ctx, &attempt); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	id := strconv.FormatUint(attempt.ID, 10)
	sent, err := provider.Send(ctx, whitelabel.SendRequest{Type: whitelabel.MessageTypeSMS, Message: input.MessageText, Encoding: encoding, Destination: attempt.Destination, ReferenceID: input.CorrelationID, Index: id, CallbackLevel: s.callbackLevel})
	if err != nil {
		code := apperrors.CodeUnknownOutcome
		var typed *apperrors.Error
		if errors.As(err, &typed) {
			code = typed.Code
		}
		attempt.State = models.SubmissionRejected
		if code == apperrors.CodeUnknownOutcome {
			attempt.State = models.SubmissionUnknown
			if parsed, parseErr := uuid.Parse(sent.MessageID); parseErr == nil && parsed != uuid.Nil {
				providerID := parsed.String()
				attempt.ProviderMessageID = &providerID
			}
		}
		slog.Warn("SMS provider submission failed", "send_id", attempt.ID, "credential_id", credentialID, "state", attempt.State, "code", code)
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
	slog.Info("SMS provider submission accepted", "send_id", attempt.ID, "credential_id", credentialID, "provider_message_id", sent.MessageID)
	s.complete(ctx, attempt)
	return []dtos.LegacySendResult{{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, IDSMS: id, CorrelationID: input.CorrelationID}}, nil
}

func (s *SMSService) complete(ctx context.Context, attempt models.SMSAttempt) {
	// Persist the final outcome even if the caller disconnected. Never resend.
	completion, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	if err := s.repository.Complete(completion, attempt); err != nil {
		// An accepted SMS remains accepted to the caller, even if the local update failed.
		slog.Error("SMS outcome persistence failed", "send_id", attempt.ID, "state", attempt.State, "provider_message_id", attempt.ProviderMessageID, "error", err)
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
