package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"gorm.io/gorm"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type credentialStub struct {
	credential models.APICredential
	err        error
}

func TestAuthenticationRejectsExpiredCredential(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{Header: make(http.Header)}
	request.SetBasicAuth("client", "test-password-only")
	for _, tc := range []struct {
		name      string
		expiresAt *time.Time
		accepted  bool
	}{
		{name: "no expiry", accepted: true},
		{name: "future", expiresAt: timePointer(time.Now().Add(time.Hour)), accepted: true},
		{name: "past", expiresAt: timePointer(time.Now().Add(-time.Hour))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewAuthService(credentialStub{credential: models.APICredential{Username: "client", PasswordHash: hash, Active: true, ExpiresAt: tc.expiresAt}})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Authenticate(context.Background(), request.Header.Get("Authorization"))
			if tc.accepted && err != nil {
				t.Fatal(err)
			}
			if !tc.accepted {
				var appError *apperrors.Error
				if !errors.As(err, &appError) || appError.Code != apperrors.CodeUnauthorized {
					t.Fatalf("err=%v", err)
				}
			}
		})
	}
}

func timePointer(value time.Time) *time.Time { return &value }

func (r credentialStub) FindByUsername(context.Context, string) (models.APICredential, error) {
	return r.credential, r.err
}

func TestAuthenticationRejectsUnknownDisabledAndWrongCredentials(t *testing.T) {
	hash, err := security.HashPassword("test-password-only")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		active   bool
		repoErr  error
		password string
		want     apperrors.Code
	}{
		{"valid", true, nil, "test-password-only", apperrors.CodeNone},
		{"wrong password", true, nil, "wrong-password", apperrors.CodeUnauthorized},
		{"disabled", false, nil, "test-password-only", apperrors.CodeUnauthorized},
		{"unknown", false, gorm.ErrRecordNotFound, "test-password-only", apperrors.CodeUnauthorized},
		{"database failure", true, errors.New("secret database error"), "test-password-only", apperrors.CodeServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewAuthService(credentialStub{credential: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: tc.active}, err: tc.repoErr})
			if err != nil {
				t.Fatal(err)
			}
			request := &http.Request{Header: make(http.Header)}
			request.SetBasicAuth("client", tc.password)
			credential, err := service.Authenticate(context.Background(), request.Header.Get("Authorization"))
			if tc.want == apperrors.CodeNone {
				if err != nil || credential.ID != 7 || credential.PasswordHash != "" {
					t.Fatalf("result=%+v err=%v", credential, err)
				}
				return
			}
			var typed *apperrors.Error
			if !errors.As(err, &typed) || typed.Code != tc.want {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
