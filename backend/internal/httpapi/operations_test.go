package httpapi

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type operationsMemory struct{ calls int }

func (r *operationsMemory) Snapshot(context.Context) (dtos.OperationsSnapshot, error) {
	r.calls++
	return dtos.OperationsSnapshot{DLR: dtos.QueueDiagnostics{Pending: 2}}, nil
}
func TestOperationsOptInAndOwnToken(t *testing.T) {
	token := strings.Repeat("o", 32)
	for _, tc := range []struct {
		enabled bool
		token   string
		want    int
		calls   int
	}{{false, "Bearer " + token, 404, 0}, {true, "", 401, 0}, {true, "Bearer callback-token", 401, 0}, {true, "Bearer " + token, 200, 1}} {
		repo := &operationsMemory{}
		router := New(nil, true, func(context.Context) error { return nil }, Dependencies{Operations: services.NewOperationsService(repo), OperationsEnabled: tc.enabled, OperationsToken: token})
		request := httptest.NewRequest(http.MethodGet, "/internal/operations", nil)
		request.Header.Set("Authorization", tc.token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.want || repo.calls != tc.calls {
			t.Fatal(response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), token) {
			t.Fatal("token exposed")
		}
	}
}
