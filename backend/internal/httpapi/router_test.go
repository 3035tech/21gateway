package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

func TestOperationalEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		fail, docs bool
		want       int
	}{
		{"liveness survives database failure", "/healthz", true, false, http.StatusOK},
		{"readiness detects database failure", "/readyz", true, false, http.StatusServiceUnavailable},
		{"readiness succeeds", "/readyz", false, false, http.StatusOK},
		{"docs disabled by default", "/openapi.json", false, false, http.StatusNotFound},
		{"openapi available when enabled", "/openapi.json", false, true, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := New([]string{"http://localhost:5173"}, tc.docs, func(context.Context) error {
				if tc.fail {
					return errors.New("private database details")
				}
				return nil
			}, Dependencies{})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.want {
				t.Fatalf("got %d want %d: %s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}

func TestReadinessErrorIsStandardizedAndPrivate(t *testing.T) {
	router := New(nil, false, func(context.Context) error { return errors.New("password=private-secret database host") }, Dependencies{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var problem dtos.ErrorOutput
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != apperrors.CodeServiceUnavailable || problem.Status != http.StatusServiceUnavailable {
		t.Fatalf("unexpected error: %+v", problem)
	}
	if problem.Detail != apperrors.CodeServiceUnavailable.Message() {
		t.Fatalf("unexpected public message: %s", problem.Detail)
	}
	if strings.Contains(response.Body.String(), "private-secret") {
		t.Fatal("internal details leaked")
	}
}

func TestDependencyDiagnosticsReportEnabledAndDisabledChecks(t *testing.T) {
	token := "diagnostics-token-with-at-least-32-bytes"
	digest := sha256.Sum256([]byte(token))
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{
		HealthDependenciesTokenSHA256: hex.EncodeToString(digest[:]),
		SMTPProbe:                     func(context.Context) error { return errors.New("password=private-secret") },
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/dependencies", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	var report dtos.DependenciesResponse
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != dtos.HealthStatusDegraded || report.Database != dtos.DependencyOK || report.SMTP != dtos.DependencyUnavailable || report.SQS != dtos.DependencyDisabled {
		t.Fatalf("unexpected report: %+v", report)
	}
	if strings.Contains(response.Body.String(), "private-secret") {
		t.Fatal("dependency diagnostic leaked internal error")
	}
}

func TestDependencyDiagnosticsHealthy(t *testing.T) {
	token := "diagnostics-token-with-at-least-32-bytes"
	digest := sha256.Sum256([]byte(token))
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{
		HealthDependenciesTokenSHA256: hex.EncodeToString(digest[:]),
		SMTPProbe:                     func(context.Context) error { return nil },
		SQSProbe:                      func(context.Context) error { return nil },
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/dependencies", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	var report dtos.DependenciesResponse
	if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != dtos.HealthStatusOK || report.Database != dtos.DependencyOK || report.SMTP != dtos.DependencyOK || report.SQS != dtos.DependencyOK {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestDependencyDiagnosticsRequireBearerBeforeProbing(t *testing.T) {
	token := "diagnostics-token-with-at-least-32-bytes"
	digest := sha256.Sum256([]byte(token))
	probes := 0
	for _, tc := range []struct {
		name, hash, header string
		want               int
	}{
		{"missing configuration", "", "Bearer " + token, http.StatusServiceUnavailable},
		{"missing token", hex.EncodeToString(digest[:]), "", http.StatusUnauthorized},
		{"incorrect token", hex.EncodeToString(digest[:]), "Bearer wrong", http.StatusUnauthorized},
		{"incorrect scheme", hex.EncodeToString(digest[:]), token, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := New(nil, false, func(context.Context) error { probes++; return nil }, Dependencies{
				HealthDependenciesTokenSHA256: tc.hash,
				SMTPProbe:                     func(context.Context) error { probes++; return nil },
			})
			request := httptest.NewRequest(http.MethodGet, "/health/dependencies", nil)
			request.Header.Set("Authorization", tc.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.want || probes != 0 {
				t.Fatalf("status=%d probes=%d, want status=%d and no probes", response.Code, probes, tc.want)
			}
		})
	}
}
