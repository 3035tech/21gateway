package services

import (
	"context"
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
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
	batchInput whitelabel.BatchRequest
	outcomes   []whitelabel.BatchResult
}

func (p *batchStub) SendBatch(_ context.Context, input whitelabel.BatchRequest) ([]whitelabel.BatchResult, error) {
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
	provider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: []whitelabel.BatchResult{{MessageID: "550e8400-e29b-41d4-a716-446655440001"}}}
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
