package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"gorm.io/gorm"
)

type submissionRows struct{}

func (submissionRows) Submission(_ context.Context, credentialID, sendID uint64) (models.SMSAttempt, error) {
	if credentialID != 7 || sendID != 42 {
		return models.SMSAttempt{}, gorm.ErrRecordNotFound
	}
	return models.SMSAttempt{ID: 42, CredentialID: 7, Destination: "5511987654321", EncryptedMessage: "private", State: models.SubmissionPending, CreatedAt: time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)}, nil
}

func TestSubmissionStatusRequiresCredentialAndHidesOtherIDs(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true}})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(nil, true, func(context.Context) error { return nil }, Dependencies{Auth: auth, Submissions: services.NewSubmissionService(submissionRows{})})
	for _, tc := range []struct {
		id, user, pass string
		want           int
	}{
		{"42", "client", "test-password-only", http.StatusOK},
		{"43", "client", "test-password-only", http.StatusNotFound},
		{"0", "client", "test-password-only", http.StatusUnprocessableEntity},
		{"42", "", "", http.StatusUnauthorized},
	} {
		request := httptest.NewRequest(http.MethodGet, "/v1/send/result?id_sms="+tc.id, nil)
		if tc.user != "" {
			request.SetBasicAuth(tc.user, tc.pass)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("id=%s status=%d body=%s", tc.id, response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "5511987654321") || strings.Contains(response.Body.String(), "private") {
			t.Fatal("private content leaked")
		}
		if tc.want == http.StatusOK {
			var status dtos.SMSSubmissionStatus
			if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil || status.IDSMS != "42" || status.State != models.SubmissionPending {
				t.Fatal("wrong submission status", err)
			}
		}
	}
}
