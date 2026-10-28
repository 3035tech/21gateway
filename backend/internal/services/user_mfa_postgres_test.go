package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/pquerna/otp/totp"
)

type recoveryMailFake struct{ tokens chan string }

func (f *recoveryMailFake) SendPasswordReset(_ context.Context, _, token string) error {
	f.tokens <- token
	return nil
}
func (f *recoveryMailFake) SendInvitation(_ context.Context, _, token string) error {
	f.tokens <- token
	return nil
}
func (f *recoveryMailFake) next(t *testing.T) string {
	t.Helper()
	select {
	case token := <-f.tokens:
		return token
	case <-time.After(2 * time.Second):
		t.Fatal("mail not delivered")
		return ""
	}
}

func TestPostgresUserMFAEnrollmentAndRecovery(t *testing.T) {
	ctx, db := postgresFixture(t)
	users, err := NewUserService(repositories.NewUsers(db))
	if err != nil {
		t.Fatal(err)
	}
	if err := users.BootstrapAdmin(ctx, "Admin", "mfa@example.test", "long-admin-password-21"); err != nil {
		t.Fatal(err)
	}
	admin, err := users.Login(ctx, "mfa@example.test", "long-admin-password-21")
	if err != nil {
		t.Fatal(err)
	}
	vault, err := security.NewTokenVault(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	mfa := NewUserMFAService(repositories.NewUserMFA(db), repositories.NewUsers(db), vault)
	users.SetMFA(mfa)
	if _, _, err := mfa.Begin(ctx, admin, "wrong-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	secret, uri, err := mfa.Begin(ctx, admin, "long-admin-password-21")
	if err != nil || secret == "" || uri == "" {
		t.Fatal(err)
	}
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	codes, err := mfa.Confirm(ctx, admin.ID, code)
	if err != nil || len(codes) != recoveryCodeCount {
		t.Fatal(err)
	}
	_, err = users.Login(ctx, admin.Email, "long-admin-password-21")
	assertCode(t, err, apperrors.CodeMFARequired)
	_, err = users.LoginWithFactor(ctx, admin.Email, "long-admin-password-21", "000000")
	assertCode(t, err, apperrors.CodeMFAInvalid)
	logged, err := users.LoginWithFactor(ctx, admin.Email, "long-admin-password-21", codes[0])
	if err != nil || logged.ID != admin.ID {
		t.Fatal(err)
	}
	_, err = users.LoginWithFactor(ctx, admin.Email, "long-admin-password-21", codes[0])
	assertCode(t, err, apperrors.CodeMFAInvalid)
	if _, err := users.SessionUser(ctx, admin.ID, admin.SessionVersion); err == nil {
		t.Fatal("enrollment did not revoke prior session")
	}
	if err := users.ChangeOwnPassword(ctx, logged, "long-admin-password-21", "a newer admin password 21"); err != nil {
		t.Fatal(err)
	}
	_, err = users.LoginWithFactor(ctx, admin.Email, "long-admin-password-21", codes[1])
	assertCode(t, err, apperrors.CodeUnauthorized)
	_, err = users.LoginWithFactor(ctx, admin.Email, "a newer admin password 21", codes[1])
	if err != nil {
		t.Fatal(err)
	}
	mailer := &recoveryMailFake{tokens: make(chan string, 10)}
	recovery := NewUserRecoveryService(repositories.NewUserRecovery(db), repositories.NewUsers(db), mfa, mailer)
	recovery.SetUserService(users)
	assertTokenLifetime := func(userID uint64, purpose string, lifetime time.Duration) {
		t.Helper()
		var token models.UserPasswordReset
		if err := db.Where("user_id = ?", userID).First(&token).Error; err != nil {
			t.Fatal(err)
		}
		remaining := time.Until(token.ExpiresAt)
		if token.Purpose != purpose || remaining < lifetime-5*time.Second || remaining > lifetime {
			t.Fatalf("purpose=%s remaining=%s, want %s", token.Purpose, remaining, lifetime)
		}
	}
	if err := recovery.Request(ctx, "unknown@example.test"); err != nil || len(mailer.tokens) != 0 {
		t.Fatal("unknown account disclosed", err)
	}
	if err := recovery.Request(ctx, admin.Email); err != nil {
		t.Fatal("recovery request", err)
	}
	resetToken := mailer.next(t)
	assertTokenLifetime(admin.ID, recoveryPurposeReset, security.PasswordResetLifetime)
	if len(resetToken) != 64 {
		t.Fatal("invalid reset token")
	}
	err = recovery.Reset(ctx, resetToken, "", "a recovered admin password 21")
	assertCode(t, err, apperrors.CodeMFARequired)
	if err := recovery.Reset(ctx, resetToken, codes[2], "a recovered admin password 21"); err != nil {
		t.Fatal(err)
	}
	err = recovery.Reset(ctx, resetToken, codes[3], "another recovered password 21")
	assertCode(t, err, apperrors.CodeInvalidResetToken)
	_, err = users.LoginWithFactor(ctx, admin.Email, "a newer admin password 21", codes[3])
	assertCode(t, err, apperrors.CodeUnauthorized)
	if _, err = users.LoginWithFactor(ctx, admin.Email, "a recovered admin password 21", codes[3]); err != nil {
		t.Fatal(err)
	}
	invited, err := recovery.Invite(ctx, admin.ID, dtos.UserBody{Name: "Second Admin", Email: "second-admin@example.test", Role: models.RoleAdmin, Active: true})
	if err != nil || invited.ID == 0 {
		t.Fatal("invite", err)
	}
	oldToken := mailer.next(t)
	assertTokenLifetime(invited.ID, recoveryPurposeInvite, security.InvitationLifetime)
	if err := recovery.ResendInvitation(ctx, invited.ID); err != nil {
		t.Fatal("resend", err)
	}
	invitationToken := mailer.next(t)
	assertTokenLifetime(invited.ID, recoveryPurposeInvite, security.InvitationLifetime)
	if invitationToken == oldToken {
		t.Fatal("invitation token reused")
	}
	err = recovery.Reset(ctx, oldToken, "", "a second admin password 21")
	assertCode(t, err, apperrors.CodeInvalidResetToken)
	if err := recovery.Reset(ctx, invitationToken, "", "a second admin password 21"); err != nil {
		t.Fatal("accept invitation", err)
	}
	if _, err := users.Login(ctx, invited.Email, "a second admin password 21"); err != nil {
		t.Fatal("second admin login", err)
	}
}
