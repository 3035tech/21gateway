package services

import (
	"context"
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"strings"
	"testing"
)

type batchMemory struct{ memoryAttempts }

func (r *batchMemory) CreateBatch(ctx context.Context, attempts []*models.SMSAttempt) error {
	if r.createErr != nil {
		return r.createErr
	}
	for _, v := range attempts {
		if err := r.Create(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
func (r *batchMemory) CompleteBatch(_ context.Context, attempts []models.SMSAttempt) error {
	r.completed = append(r.completed, attempts...)
	return r.completeErr
}

type batchStub struct {
	stubProvider
	calls      int
	batchInput smsprovider.BatchRequest
	outcomes   []smsprovider.BatchResult
}

func (p *batchStub) SendBatch(_ context.Context, input smsprovider.BatchRequest) ([]smsprovider.BatchResult, error) {
	p.calls++
	p.batchInput = input
	return p.outcomes, p.err
}

type countLimiter struct {
	count   int
	allowed bool
}

func (l *countLimiter) Allow(_ context.Context, _ uint64, count int) (bool, error) {
	l.count = count
	return l.allowed, nil
}
func TestBatchRejectsInvalidDestinationBeforeProvider(t *testing.T) {
	repo := &batchMemory{}
	provider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: []smsprovider.BatchResult{{MessageID: "550e8400-e29b-41d4-a716-446655440001"}}}
	limiter := &countLimiter{allowed: true}
	invalid := validSMS()
	invalid.Destination = "invalid"
	results, err := NewSMSServiceWithLimiter(repo, provider, limiter).Send(context.Background(), 1, []dtos.LegacySMS{validSMS(), invalid})
	if err != nil || len(results) != 2 || results[0].IDSMS != "1" || results[1].Message != dtos.LegacyInvalidNumber || provider.calls != 1 || limiter.count != 2 {
		t.Fatal("batch translation failed")
	}
	if len(provider.batchInput.Messages) != 1 || provider.batchInput.Messages[0].Destination != "5511987654321" || len(repo.completed) != 1 {
		t.Fatal("invalid destination reached provider or persistence")
	}
}

func TestBatchUsesLatin1AndRecomputesEncodingAfterLengthRejection(t *testing.T) {
	latinRepo := &batchMemory{}
	latinProvider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: []smsprovider.BatchResult{{MessageID: "550e8400-e29b-41d4-a716-446655440001"}, {MessageID: "550e8400-e29b-41d4-a716-446655440002"}}}
	latinMessages := []dtos.LegacySMS{validSMS(), validSMS()}
	latinMessages[0].MessageText = "×"
	if _, err := NewSMSService(latinRepo, latinProvider).Send(context.Background(), 1, latinMessages); err != nil {
		t.Fatal(err)
	}
	if latinProvider.batchInput.Encoding != models.SMSEncodingLatin1 {
		t.Fatalf("batch did not select Latin1: %q", latinProvider.batchInput.Encoding)
	}

	repo := &batchMemory{}
	provider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: []smsprovider.BatchResult{{MessageID: "550e8400-e29b-41d4-a716-446655440003"}}}
	messages := []dtos.LegacySMS{validSMS(), validSMS()}
	messages[0].MessageText = strings.Repeat("×", 71)
	result, err := NewSMSService(repo, provider).Send(context.Background(), 1, messages)
	if err != nil || result[0].Message != dtos.LegacyInvalidMessage || result[1].Status != dtos.LegacyStatusOK {
		t.Fatalf("overlength item handling failed: result=%+v err=%v", result, err)
	}
	if provider.batchInput.Encoding != models.SMSEncodingGSM7 || len(provider.batchInput.Messages) != 1 || provider.batchInput.Messages[0].Message != validSMS().MessageText || len(repo.completed) != 1 {
		t.Fatalf("valid remainder was not sent with its own global encoding: request=%+v completed=%+v", provider.batchInput, repo.completed)
	}
}
func TestBatchFailureNeverRetriesAndPersistencePrecedesSend(t *testing.T) {
	for _, failCreate := range []bool{true, false} {
		repo := &batchMemory{}
		provider := &batchStub{stubProvider: stubProvider{enabled: true, err: apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)}}
		if failCreate {
			repo.createErr = errors.New("private database failure")
		}
		result, err := NewSMSService(repo, provider).Send(context.Background(), 1, []dtos.LegacySMS{validSMS(), validSMS()})
		if failCreate {
			if err == nil || provider.calls != 0 {
				t.Fatal("batch sent without persistence")
			}
		} else if err != nil || provider.calls != 1 || result[0].IDSMS == "" || result[1].Message != dtos.LegacyUnknownOutcome || repo.completed[0].State != models.SubmissionUnknown {
			t.Fatal("uncertain batch acceptance retried or lost")
		}
	}
}
