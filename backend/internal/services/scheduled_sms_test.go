package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type scheduleMemory struct {
	jobs []models.ScheduledSMS
	err  error
}

func (r *scheduleMemory) Enqueue(_ context.Context, jobs []models.ScheduledSMS) ([]models.ScheduledSMS, error) {
	if r.err != nil {
		return nil, r.err
	}
	for i := range jobs {
		jobs[i].ID = uint64(i + 1)
		jobs[i].AttemptID = uint64(i + 1001)
		jobs[i].Attempt.ID = jobs[i].AttemptID
	}
	r.jobs = jobs
	return jobs, nil
}
func scheduledVault(t testing.TB) *security.ScheduledContentVault {
	t.Helper()
	v, err := security.NewScheduledContentVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func futureSMS() dtos.LegacySMS {
	input := validSMS()
	at := time.Now().Add(time.Hour).Unix()
	input.ScheduledDate = &at
	input.MessageText = "Olá, ação! 😊"
	return input
}
func TestAllSendsAreAcceptedBeforeProviderCallInQueueMode(t *testing.T) {
	repo := &scheduleMemory{}
	provider := &stubProvider{enabled: true}
	limiter := &countLimiter{allowed: true}
	service := NewSMSServiceWithLimiter(&memoryAttempts{}, provider, limiter)
	service.SetDispatchQueue(NewScheduledSMSService(repo, scheduledVault(t)))
	inputs := []dtos.LegacySMS{validSMS(), futureSMS()}
	results, err := service.Send(context.Background(), 7, inputs)
	if err != nil || len(results) != 2 || len(repo.jobs) != 2 || len(provider.inputs) != 0 || limiter.count != 0 {
		t.Fatal("queue admission called provider or lost receipt", err)
	}
	for i, result := range results {
		if result.IDSMS != strconv.FormatUint(repo.jobs[i].AttemptID, 10) || result.Code != dtos.LegacyCodeAccepted || repo.jobs[i].Attempt.State != models.SubmissionPending {
			t.Fatal("incorrect queued receipt")
		}
	}
	if repo.jobs[0].Attempt.ScheduledAt != nil || repo.jobs[1].Attempt.ScheduledAt == nil || repo.jobs[0].DueAt.After(time.Now()) {
		t.Fatal("incorrect immediate/future due time")
	}
	if text, err := scheduledVault(t).Decrypt(7, repo.jobs[0].EventKey, repo.jobs[0].EncryptedMessage); err != nil || text != inputs[0].MessageText {
		t.Fatal("queued content encryption")
	}
}
func TestScheduledAdmissionEncrypts200AndReservesNoProviderBudget(t *testing.T) {
	repo := &scheduleMemory{}
	vault := scheduledVault(t)
	historyVault, err := security.NewSMSHistoryVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("h", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	budget := &countLimiter{allowed: true}
	service := NewSMSServiceWithLimiter(&memoryAttempts{}, nil, budget)
	scheduler := NewScheduledSMSService(repo, vault)
	scheduler.SetHistoryVault(historyVault)
	service.SetScheduler(scheduler)
	inputs := make([]dtos.LegacySMS, models.ScheduledBatchSize)
	for i := range inputs {
		inputs[i] = futureSMS()
		inputs[i].CorrelationID = fmt.Sprintf("ref-%d", i)
	}
	result, err := service.Send(context.Background(), 7, inputs)
	if err != nil || len(result) != models.ScheduledBatchSize || budget.count != 0 || len(repo.jobs) != models.ScheduledBatchSize {
		t.Fatal("scheduled batch admission", err)
	}
	for i, job := range repo.jobs {
		if result[i].IDSMS != strconv.FormatUint(job.AttemptID, 10) || result[i].CorrelationID != inputs[i].CorrelationID || result[i].Code != dtos.LegacyCodeAccepted || job.Attempt.ProviderMessageID != nil || job.Attempt.ScheduledAt == nil {
			t.Fatal("scheduled receipt/metadata", i)
		}
		plain, err := vault.Decrypt(7, job.EventKey, job.EncryptedMessage)
		if err != nil || plain != "Ola, acao!" || strings.Contains(job.EncryptedMessage, "Ola, acao!") {
			t.Fatal("content protection", err)
		}
		if job.Attempt.ContentRef == nil || job.Attempt.EncryptedMessage == "" {
			t.Fatal("history content missing")
		}
		historyText, err := historyVault.Decrypt(7, *job.Attempt.ContentRef, job.Attempt.EncryptedMessage)
		if err != nil || historyText != plain {
			t.Fatal("scheduled history differs", err)
		}
	}
}
func TestScheduledMixedRequestPreservesOrderAndQueueFailureDoesNotSend(t *testing.T) {
	for _, failQueue := range []bool{false, true} {
		t.Run(fmt.Sprint(failQueue), func(t *testing.T) {
			repo := &scheduleMemory{}
			if failQueue {
				repo.err = errors.New("private SQL")
			}
			provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440001", Status: whitelabel.StatusSent}}
			limiter := &countLimiter{allowed: true}
			service := NewSMSServiceWithLimiter(&memoryAttempts{}, provider, limiter)
			service.SetScheduler(NewScheduledSMSService(repo, scheduledVault(t)))
			result, err := service.Send(context.Background(), 7, []dtos.LegacySMS{validSMS(), futureSMS()})
			if failQueue {
				if err == nil || len(provider.inputs) != 0 {
					t.Fatal("queue failure followed live send")
				}
				return
			}
			if err != nil || result[0].IDSMS != "1" || result[1].IDSMS != "1001" || len(provider.inputs) != 1 || limiter.count != 1 {
				t.Fatal("mixed order or budget", err)
			}
		})
	}
}
func TestScheduledAdmissionRejectsPastInvalidDateAndPhone(t *testing.T) {
	for _, change := range []func(*dtos.LegacySMS){func(m *dtos.LegacySMS) { at := time.Now().Add(-time.Hour).Unix(); m.ScheduledDate = &at }, func(m *dtos.LegacySMS) { at := maximumScheduleTimestamp + 1; m.ScheduledDate = &at }, func(m *dtos.LegacySMS) { m.Destination = "1199999999" }} {
		repo := &scheduleMemory{}
		m := futureSMS()
		change(&m)
		result, err := NewScheduledSMSService(repo, scheduledVault(t)).Enqueue(context.Background(), 7, []dtos.LegacySMS{m})
		if err != nil || result[0].Status != dtos.LegacyStatusNOK || len(repo.jobs) != 0 {
			t.Fatal("invalid schedule persisted", err)
		}
	}
}

type scheduledWorkerMemory struct {
	jobs                      []models.ScheduledSMS
	credential                models.APICredential
	begun, released, rejected int
	completed                 []models.SMSAttempt
	finishErr                 error
}

func (r *scheduledWorkerMemory) Claim(context.Context) ([]models.ScheduledSMS, error) {
	jobs := r.jobs
	r.jobs = nil
	return jobs, nil
}
func (r *scheduledWorkerMemory) Credential(context.Context, uint64) (models.APICredential, error) {
	return r.credential, nil
}
func (r *scheduledWorkerMemory) Begin(context.Context, []models.ScheduledSMS) error {
	r.begun++
	return nil
}
func (r *scheduledWorkerMemory) Release(context.Context, []models.ScheduledSMS) error {
	r.released++
	return nil
}
func (r *scheduledWorkerMemory) Reject(context.Context, []models.ScheduledSMS, apperrors.Code) error {
	r.rejected++
	return nil
}
func (r *scheduledWorkerMemory) Finish(_ context.Context, _ []models.ScheduledSMS, attempts []models.SMSAttempt) error {
	r.completed = attempts
	return r.finishErr
}
func TestScheduledWorkerBatchReusesIDsAndDoesNotRetryUnknown(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			queue := &scheduleMemory{}
			vault := scheduledVault(t)
			inputs := make([]dtos.LegacySMS, 200)
			outcomes := make([]whitelabel.BatchResult, 200)
			for i := range inputs {
				inputs[i] = futureSMS()
				outcomes[i].MessageID = fmt.Sprintf("550e8400-e29b-41d4-a716-%012d", i+1)
			}
			if _, err := NewScheduledSMSService(queue, vault).Enqueue(context.Background(), 7, inputs); err != nil {
				t.Fatal(err)
			}
			repo := &scheduledWorkerMemory{jobs: queue.jobs, credential: models.APICredential{ID: 7, Active: true}}
			provider := &batchStub{stubProvider: stubProvider{enabled: true}, outcomes: outcomes}
			if unknown {
				provider.err = apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
			}
			worker := NewScheduledSMSWorker(repo, provider, &countLimiter{allowed: true}, vault)
			if err := worker.process(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := worker.process(context.Background()); err != nil {
				t.Fatal(err)
			}
			if provider.calls != 1 || repo.begun != 1 || len(repo.completed) != 200 || repo.released != 0 {
				t.Fatal("worker retried or lost batch")
			}
			for i, a := range repo.completed {
				want := models.SubmissionAccepted
				if unknown {
					want = models.SubmissionUnknown
				}
				if a.ID != queue.jobs[i].AttemptID || a.State != want || provider.batchInput.Messages[i].Index != strconv.FormatUint(a.ID, 10) {
					t.Fatal("stable ID/outcome", i)
				}
			}
		})
	}
}
func TestScheduledWorkerDefersBeforeSendAndBlocksInactiveAccount(t *testing.T) {
	for _, active := range []bool{false, true} {
		queue := &scheduleMemory{}
		vault := scheduledVault(t)
		if _, err := NewScheduledSMSService(queue, vault).Enqueue(context.Background(), 7, []dtos.LegacySMS{futureSMS()}); err != nil {
			t.Fatal(err)
		}
		repo := &scheduledWorkerMemory{jobs: queue.jobs, credential: models.APICredential{ID: 7, Active: active}}
		provider := &stubProvider{enabled: true}
		if err := NewScheduledSMSWorker(repo, provider, &countLimiter{allowed: false}, vault).process(context.Background()); err != nil {
			t.Fatal(err)
		}
		if len(provider.inputs) != 0 || repo.begun != 0 {
			t.Fatal("blocked job reached provider")
		}
		if active && repo.released != 1 || !active && repo.rejected != 1 {
			t.Fatal("incorrect blocked job disposition")
		}
	}
}
func BenchmarkScheduledAdmission200(b *testing.B) {
	vault := scheduledVault(b)
	inputs := make([]dtos.LegacySMS, 200)
	for i := range inputs {
		inputs[i] = futureSMS()
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := NewScheduledSMSService(&scheduleMemory{}, vault).Enqueue(context.Background(), 7, inputs); err != nil {
			b.Fatal(err)
		}
	}
}

type partialScheduleBudget struct {
	granted  int
	reserves int
	checks   int
}

func (b *partialScheduleBudget) Allow(context.Context, uint64, int) (bool, error) {
	b.checks++
	return false, nil
}
func (b *partialScheduleBudget) ReserveUpTo(context.Context, uint64, int) (int, error) {
	b.reserves++
	return b.granted, nil
}
func TestScheduledDispatchUsesAvailableCapacityWithoutWaitingForWholeBatch(t *testing.T) {
	queue := &scheduleMemory{}
	vault := scheduledVault(t)
	if _, err := NewScheduledSMSService(queue, vault).Enqueue(context.Background(), 7, []dtos.LegacySMS{futureSMS(), futureSMS(), futureSMS()}); err != nil {
		t.Fatal(err)
	}
	repo := &scheduledWorkerMemory{credential: models.APICredential{ID: 7, Active: true}}
	provider := &stubProvider{enabled: true, response: whitelabel.SendResponse{MessageID: "550e8400-e29b-41d4-a716-446655440099", Status: whitelabel.StatusSent}}
	budget := &partialScheduleBudget{granted: 1}
	progress, err := NewScheduledSMSWorker(repo, provider, budget, vault).processGroup(context.Background(), queue.jobs)
	if err != nil || !progress || len(provider.inputs) != 1 || len(repo.completed) != 1 || repo.completed[0].ID != queue.jobs[0].AttemptID || repo.released != 1 || budget.reserves != 1 || budget.checks != 0 {
		t.Fatal("partial budget stalled, duplicated reservation or lost receipt", err)
	}
}
