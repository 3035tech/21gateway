package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
)

type scheduledHTTPQueue struct{ jobs []models.ScheduledSMS }

func (r *scheduledHTTPQueue) Enqueue(_ context.Context, jobs []models.ScheduledSMS) ([]models.ScheduledSMS, error) {
	for i := range jobs {
		jobs[i].AttemptID = uint64(i + 42)
	}
	r.jobs = jobs
	return jobs, nil
}
func TestScheduledSMSHTTPAcceptsWithoutContactingProvider(t *testing.T) {
	hash, err := security.HashPassword("existing-api-password")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", Active: true, PasswordHash: hash}})
	if err != nil {
		t.Fatal(err)
	}
	vault, err := security.NewScheduledContentVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	queue := &scheduledHTTPQueue{}
	attempts := &testAttempts{}
	sender := services.NewSMSService(attempts, nil)
	sender.SetScheduler(services.NewScheduledSMSService(queue, vault))
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{Auth: auth, SMS: sender})
	body := fmt.Sprintf(`{"sms":[{"messageText":"Olá, ação!","destination":"11987654321","correlationId":"scheduled-ref","scheduledDate":%d}]}`, time.Now().Add(time.Hour).Unix())
	request := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(body))
	request.SetBasicAuth("client", "existing-api-password")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	var result []dtos.LegacySendResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(result) != 1 || result[0].IDSMS != "42" || result[0].CorrelationID != "scheduled-ref" || result[0].Code != dtos.LegacyCodeAccepted || len(queue.jobs) != 1 || len(attempts.created) != 0 {
		t.Fatal("scheduled HTTP receipt", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), queue.jobs[0].EncryptedMessage) {
		t.Fatal("ciphertext leaked")
	}
}
