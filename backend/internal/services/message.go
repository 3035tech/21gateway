package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

const testSMSMessage = "Teste de integracao 21 Mobile."
const testSMSCompanyPrefix = "Teste do token Zoug da empresa "
const testSendAuditAction = "test_send_requested"

type MessageService struct {
	repository              *repositories.SMSAttempts
	credentials             *repositories.Credentials
	audit                   *repositories.Audits
	sender                  *SMSService
	testDestination         string
	testAllowAnyDestination bool
	history                 *security.SMSHistoryVault
}

func NewMessageService(repo *repositories.SMSAttempts) *MessageService {
	return &MessageService{repository: repo}
}
func NewMessageServiceWithTestSend(repo *repositories.SMSAttempts, credentials *repositories.Credentials, audit *repositories.Audits, sender *SMSService, destination string, allowAnyDestination bool) *MessageService {
	return &MessageService{repository: repo, credentials: credentials, audit: audit, sender: sender, testDestination: destination, testAllowAnyDestination: allowAnyDestination}
}
func (s *MessageService) WithHistoryVault(vault *security.SMSHistoryVault) *MessageService {
	s.history = vault
	return s
}

const SMSHistoryRetentionDays = 90

func (s *MessageService) Content(ctx context.Context, id uint64) (dtos.MessageContentView, error) {
	attempt, err := s.repository.MessageContent(ctx, id)
	if err != nil {
		return dtos.MessageContentView{}, dbError(err)
	}
	retentionStart := attempt.CreatedAt
	if attempt.ScheduledAt != nil {
		retentionStart = *attempt.ScheduledAt
	}
	if attempt.DispatchedAt != nil {
		retentionStart = *attempt.DispatchedAt
	}
	if attempt.ContentPurgedAt != nil || time.Now().UTC().After(retentionStart.Add(SMSHistoryRetentionDays*24*time.Hour)) {
		return dtos.MessageContentView{State: dtos.MessageContentExpired}, nil
	}
	if attempt.EncryptedMessage == "" || attempt.ContentRef == nil {
		return dtos.MessageContentView{State: dtos.MessageContentUnavailable}, nil
	}
	if !s.history.Enabled() {
		return dtos.MessageContentView{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	message, err := s.history.Decrypt(attempt.CredentialID, *attempt.ContentRef, attempt.EncryptedMessage)
	if err != nil {
		return dtos.MessageContentView{}, apperrors.Wrap(apperrors.CodeInternal, err)
	}
	return dtos.MessageContentView{State: dtos.MessageContentAvailable, MessageText: message}, nil
}
func (s *MessageService) Messages(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.MessageView], error) {
	values, err := s.repository.Messages(ctx, q)
	return values, dbError(err)
}
func (s *MessageService) SendTest(ctx context.Context, actor, credentialID uint64, destination, messageText string) (dtos.TestSendResult, error) {
	var empty dtos.TestSendResult
	if (!s.testAllowAnyDestination && s.testDestination == "") || s.credentials == nil || s.sender == nil || s.audit == nil {
		return empty, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	if !brazilLocalNumber.MatchString(destination) {
		return empty, apperrors.Wrap(apperrors.CodeInvalidDestination, nil)
	}
	if !s.testAllowAnyDestination && destination != s.testDestination {
		return empty, apperrors.Wrap(apperrors.CodeTestDestinationNotAllowed, nil)
	}
	credential, err := s.credentials.ForTestSend(ctx, credentialID)
	if err != nil {
		return empty, dbError(err)
	}
	if !credential.ValidAt(time.Now()) || (credential.CompanyID != nil && (credential.CompanyActive == nil || !*credential.CompanyActive)) {
		return empty, apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	credential.PasswordHash = ""
	if strings.TrimSpace(messageText) == "" {
		if messageText != "" {
			return empty, apperrors.Wrap(apperrors.CodeInvalidMessage, nil)
		}
		messageText = standardTestMessage(credential.CompanyName)
	}
	if code := validateMessage(dtos.LegacySMS{Destination: destination, MessageText: messageText}); code != apperrors.CodeNone {
		return empty, apperrors.Wrap(code, nil)
	}
	if err := s.audit.CreateEvent(ctx, &actor, testSendAuditAction, string(security.ResourceCredentials), &credentialID); err != nil {
		return empty, dbError(err)
	}
	correlationID := "test-" + uuid.NewString()
	results, err := s.sender.Send(WithCredential(ctx, credential), credentialID, []dtos.LegacySMS{{Destination: destination, MessageText: messageText, CorrelationID: correlationID}})
	if err != nil {
		return empty, err
	}
	result := results[0]
	state := models.SubmissionRejected
	if result.Status == dtos.LegacyStatusOK {
		state = models.SubmissionAccepted
	} else if result.Message == dtos.LegacyUnknownOutcome {
		state = models.SubmissionUnknown
	}
	return dtos.TestSendResult{SubmissionState: state, IDSMS: result.IDSMS, CorrelationID: correlationID, FailureCode: string(result.Message)}, nil
}

func standardTestMessage(companyName string) string {
	companyName = strings.TrimSpace(companyName)
	if companyName == "" {
		return testSMSMessage
	}
	maximumNameLength := MaximumMessageCharacters - utf8.RuneCountInString(testSMSCompanyPrefix) - 1
	name := []rune(companyName)
	if len(name) > maximumNameLength {
		name = name[:maximumNameLength]
	}
	return testSMSCompanyPrefix + string(name) + "."
}
