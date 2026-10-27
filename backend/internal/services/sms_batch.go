package services

import (
	"context"
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"log/slog"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maximumRecordedDestinationBytes = 128

type BatchProvider interface {
	SendBatch(context.Context, whitelabel.BatchRequest) ([]whitelabel.BatchResult, error)
}
type BatchAttemptRepository interface {
	CreateBatch(context.Context, []*models.SMSAttempt) error
	CompleteBatch(context.Context, []models.SMSAttempt) error
}

func (s *SMSService) sendBatch(ctx context.Context, credentialID uint64, messages []dtos.LegacySMS) ([]dtos.LegacySendResult, error) {
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
	batch, ok := provider.(BatchProvider)
	if !ok || !provider.Enabled() {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	repository, ok := s.repository.(BatchAttemptRepository)
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	allowed, err := s.allow(ctx, credentialID, len(messages))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !allowed {
		result := make([]dtos.LegacySendResult, len(messages))
		for i, input := range messages {
			result[i] = failure(input, apperrors.CodeRateLimited)
		}
		return result, nil
	}
	messages = append([]dtos.LegacySMS(nil), messages...)
	result := make([]dtos.LegacySendResult, len(messages))
	attempts := make([]*models.SMSAttempt, 0, len(messages))
	positions := make([]int, 0, len(messages))
	encoding := models.SMSEncodingGSM7
	for i, input := range messages {
		code := apperrors.CodeNone
		normalized, normalizationCode := prepareSMSText(input.MessageText)
		// Rules not representable by the supplier must be rejected locally, never silently sent immediately.
		if input.ScheduledDate != nil && *input.ScheduledDate != 0 {
			code = apperrors.CodeScheduleUnsupported
		} else if normalizationCode != apperrors.CodeNone {
			code = apperrors.CodeInvalidMessage
		} else if utf8.RuneCountInString(input.CorrelationID) > MaximumCorrelationCharacters {
			code = apperrors.CodeInvalidRequest
		}
		// Ten-digit normalization still needs an example from the legacy implementation.
		if len(input.Destination) > maximumRecordedDestinationBytes || (len(input.Destination) == 10 && allDigits(input.Destination)) {
			code = apperrors.CodeInvalidDestination
		}
		if code != apperrors.CodeNone {
			result[i] = failure(input, code)
			continue
		}
		input.MessageText = normalized
		messages[i] = input
		destination := input.Destination
		if brazilLocalNumber.MatchString(destination) {
			destination = brazilCountryCode + destination
		}
		if whitelabel.SelectEncoding(input.MessageText) == models.SMSEncodingUCS2 {
			encoding = models.SMSEncodingUCS2
		}
		attempt := &models.SMSAttempt{CredentialID: credentialID, Destination: destination, CorrelationID: input.CorrelationID, State: models.SubmissionPending}
		if credential, ok := CredentialFromContext(ctx); ok && credential.ID == credentialID {
			attempt.CompanyID = credential.CompanyID
		}
		attempts = append(attempts, attempt)
		positions = append(positions, i)
	}
	if len(attempts) == 0 {
		return result, nil
	}
	for _, attempt := range attempts {
		attempt.Encoding = encoding
	}
	if err := repository.CreateBatch(ctx, attempts); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	request := whitelabel.BatchRequest{Type: whitelabel.MessageTypeSMS, Encoding: encoding, Messages: make([]whitelabel.BatchMessage, len(attempts))}
	for i, attempt := range attempts {
		input := messages[positions[i]]
		request.Messages[i] = whitelabel.BatchMessage{Message: input.MessageText, Destination: attempt.Destination, ReferenceID: input.CorrelationID, Index: strconv.FormatUint(attempt.ID, 10)}
	}
	outcomes, err := batch.SendBatch(ctx, request)
	if err == nil && len(outcomes) != len(attempts) {
		err = apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
	}
	completed := make([]models.SMSAttempt, 0, len(attempts))
	for i, attempt := range attempts {
		code := apperrors.CodeNone
		id := strconv.FormatUint(attempt.ID, 10)
		input := messages[positions[i]]
		if err != nil {
			code = apperrors.CodeUnknownOutcome
			var typed *apperrors.Error
			if errors.As(err, &typed) {
				code = typed.Code
			}
		} else {
			code = outcomes[i].ErrorCode
			if code == apperrors.CodeNone && outcomes[i].MessageID == "" {
				code = apperrors.CodeUnknownOutcome
			}
		}
		if code == apperrors.CodeNone {
			attempt.State = models.SubmissionAccepted
			attempt.ProviderMessageID = &outcomes[i].MessageID
			result[positions[i]] = dtos.LegacySendResult{Status: dtos.LegacyStatusOK, Code: dtos.LegacyCodeAccepted, IDSMS: id, CorrelationID: input.CorrelationID}
		} else {
			attempt.State = models.SubmissionRejected
			if code == apperrors.CodeUnknownOutcome {
				attempt.State = models.SubmissionUnknown
			}
			attempt.FailureCode = code
			result[positions[i]] = failure(input, code)
			if attempt.State == models.SubmissionUnknown {
				result[positions[i]].IDSMS = id
			}
		}
		completed = append(completed, *attempt)
	}
	completion, cancel := context.WithTimeout(context.WithoutCancel(ctx), completionTimeout)
	defer cancel()
	if err := repository.CompleteBatch(completion, completed); err != nil {
		slog.Error("Batch outcome persistence failed", "items", len(completed))
	}
	return result, nil
}
func allDigits(value string) bool {
	return value != "" && strings.IndexFunc(value, func(r rune) bool { return r < '0' || r > '9' }) < 0
}
