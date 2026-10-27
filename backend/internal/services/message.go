package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/google/uuid"
	"time"
)

const testSMSMessage = "Teste de integracao 21 Mobile."
const testSendAuditAction = "test_send_requested"

type MessageService struct {
	repository      *repositories.SMSAttempts
	credentials     *repositories.Credentials
	audit           *repositories.Audits
	sender          *SMSService
	testDestination string
}

func NewMessageService(repo *repositories.SMSAttempts) *MessageService {
	return &MessageService{repository: repo}
}
func NewMessageServiceWithTestSend(repo *repositories.SMSAttempts, credentials *repositories.Credentials, audit *repositories.Audits, sender *SMSService, destination string) *MessageService {
	return &MessageService{repository: repo, credentials: credentials, audit: audit, sender: sender, testDestination: destination}
}
func (s *MessageService) Messages(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.MessageView], error) {
	values, err := s.repository.Messages(ctx, q)
	return values, dbError(err)
}
func (s *MessageService) SendTest(ctx context.Context, actor, credentialID uint64, destination string) (dtos.TestSendResult, error) {
	var empty dtos.TestSendResult
	if s.testDestination == "" || s.credentials == nil || s.sender == nil || s.audit == nil {
		return empty, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	if destination != s.testDestination {
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
	if err := s.audit.CreateEvent(ctx, &actor, testSendAuditAction, string(security.ResourceCredentials), &credentialID); err != nil {
		return empty, dbError(err)
	}
	correlationID := "test-" + uuid.NewString()
	results, err := s.sender.Send(WithCredential(ctx, credential), credentialID, []dtos.LegacySMS{{Destination: destination, MessageText: testSMSMessage, CorrelationID: correlationID}})
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
