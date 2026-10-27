package services

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"gorm.io/gorm"
)

func TestPostgresSecurityAuthenticationAndInjection(t *testing.T) {
	ctx, db := postgresFixture(t)
	users, err := NewUserService(repositories.NewUsers(db))
	if err != nil {
		t.Fatal(err)
	}
	password := "a long controlled test passphrase"
	if err := users.BootstrapAdmin(ctx, "Assessment", "admin@example.test", password); err != nil {
		t.Fatal(err)
	}
	protection := repositories.NewAuthenticationAttempts(db)
	key := security.AuthenticationKey("admin-account", "admin@example.test")
	blocked := false
	// Argon2 checks can cross a PostgreSQL minute boundary during this test.
	for i := 0; i < 2*security.AuthenticationFailureLimit; i++ {
		_, err := users.Login(ctx, " ADMIN@example.test ", "incorrect-password")
		assertCode(t, err, apperrors.CodeUnauthorized)
		blocked, err = protection.Blocked(ctx, key, security.AuthenticationFailureLimit)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
	}
	if !blocked {
		t.Fatal("authentication failures did not reach the shared limit")
	}
	_, err = users.Login(ctx, "admin@example.test", password)
	assertCode(t, err, apperrors.CodeRateLimited)
	if err := db.Exec("UPDATE login_limits SET window_start=? WHERE key=?", time.Now().Add(-time.Minute), key).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := users.Login(ctx, "admin@example.test", password); err != nil {
		t.Fatal("temporary lock did not expire", err)
	}
	var allowed atomic.Int32
	var group sync.WaitGroup
	failures := make(chan error, 20)
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			ok, err := protection.Reserve(ctx, "assessment-concurrency", 10)
			if err != nil {
				failures <- err
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if allowed.Load() != 10 {
		t.Fatal("shared atomic budget", allowed.Load())
	}
	// Actual production repositories: hostile values must not broaden authentication/filtering.
	for _, payload := range []string{"' OR '1'='1' --", "'; SELECT pg_sleep(2); --", "admin' UNION SELECT NULL --"} {
		if _, err := repositories.NewUsers(db).UserByEmail(ctx, payload); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatal("injected email matched", err)
		}
		if _, err := repositories.NewCredentials(db).FindByUsername(ctx, payload); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatal("injected username matched", err)
		}
		page, err := repositories.NewUsers(db).Users(ctx, dtos.ListQuery{Page: 1, PageSize: 10, Search: payload, Direction: dtos.SortAscending})
		if err != nil || page.Total != 0 {
			t.Fatal("injected search matched", err)
		}
	}
	_, err = repositories.NewUsers(db).Users(ctx, dtos.ListQuery{Page: 1, PageSize: 10, Sort: "id; SELECT pg_sleep(2)", Direction: dtos.SortAscending})
	assertCode(t, err, apperrors.CodeInvalidRequest)
	// A legacy weak password continues to authenticate: never silently replace client secrets.
	hash, err := security.HashPassword("old")
	if err != nil {
		t.Fatal(err)
	}
	credential := models.APICredential{Username: "legacy", PasswordHash: hash, Active: true}
	if err := db.Create(&credential).Error; err != nil {
		t.Fatal(err)
	}
	auth, err := NewAuthServiceWithProtection(repositories.NewCredentials(db), protection, security.APIAuthenticationIPLimit)
	if err != nil {
		t.Fatal(err)
	}
	request := &http.Request{Header: make(http.Header)}
	request.SetBasicAuth("legacy", "old")
	if _, err := auth.AuthenticateRequest(ctx, request.Header.Get("Authorization"), "192.0.2.1:1234"); err != nil {
		t.Fatal("legacy password changed", err)
	}
	var count int64
	if err := db.Model(&models.SystemUser{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("system users changed", err)
	}

}
