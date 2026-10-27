package services

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"gorm.io/gorm"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type credentialStub struct {
	credential models.APICredential
	err        error
}

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
