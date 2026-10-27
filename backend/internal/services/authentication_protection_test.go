package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type failingProtection struct{ stage string }

func (f failingProtection) Reserve(context.Context, string, int) (bool, error) {
	if f.stage == "reserve" {
		return false, errors.New("private limiter DB")
	}
	return true, nil
}
func (f failingProtection) Blocked(context.Context, string, int) (bool, error) {
	if f.stage == "blocked" {
		return false, errors.New("private limiter DB")
	}
	return false, nil
}
func (f failingProtection) Failure(context.Context, string, int) error {
	if f.stage == "failure" {
		return errors.New("private limiter DB")
	}
	return nil
}

type countedCredentials struct {
	calls      int
	credential models.APICredential
}

func (r *countedCredentials) FindByUsername(context.Context, string) (models.APICredential, error) {
	r.calls++
	return r.credential, nil
}
func TestAuthenticationProtectionFailsClosed(t *testing.T) {
	hash, err := security.HashPassword("existing-legacy-password")
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{Header: make(http.Header)}
	request.SetBasicAuth("client", "wrong-password")
	for _, stage := range []string{"reserve", "blocked", "failure"} {
		t.Run(stage, func(t *testing.T) {
			repo := &countedCredentials{credential: models.APICredential{ID: 7, Active: true, PasswordHash: hash}}
			auth, err := NewAuthServiceWithProtection(repo, failingProtection{stage: stage}, security.APIAuthenticationIPLimit)
			if err != nil {
				t.Fatal(err)
			}
			_, err = auth.AuthenticateRequest(context.Background(), request.Header.Get("Authorization"), "192.0.2.1:1234")
			assertCode(t, err, apperrors.CodeServiceUnavailable)
			if stage != "failure" && repo.calls != 0 {
				t.Fatal("failed protection reached account lookup")
			}
		})
	}
}
func TestAuthenticationCPUConcurrencyIsBounded(t *testing.T) {
	repo := &countedCredentials{}
	auth, err := NewAuthService(repo)
	if err != nil {
		t.Fatal(err)
	}
	for range security.MaxConcurrentPasswordChecks {
		auth.checks <- struct{}{}
	}
	request := &http.Request{Header: make(http.Header)}
	request.SetBasicAuth("client", "guess")
	_, err = auth.Authenticate(context.Background(), request.Header.Get("Authorization"))
	assertCode(t, err, apperrors.CodeServiceUnavailable)
	if repo.calls != 0 {
		t.Fatal("saturated verifier reached repository")
	}
}
