package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/database"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
)

type action string

const (
	actionMigrate          action = "migrate"
	actionBootstrapAdmin   action = "bootstrap-admin"
	actionCreateCredential action = "create-credential"
	adminTimeout                  = 30 * time.Second
)

func run() error {
	operation := flag.String("action", string(actionMigrate), "migrate or create-credential")
	username := flag.String("username", "", "API username for create-credential")
	name := flag.String("name", "", "Name of first administrator")
	email := flag.String("email", "", "Email of first administrator")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), adminTimeout)
	defer cancel()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeDatabaseUnavailable, err)
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	switch action(*operation) {
	case actionMigrate:
		if err := database.Migrate(db.WithContext(ctx)); err != nil {
			return apperrors.Wrap(apperrors.CodeDatabaseUnavailable, err)
		}
		fmt.Println("Migrations applied.")
	case actionBootstrapAdmin:
		service, err := services.NewUserService(repositories.NewUsers(db))
		if err != nil {
			return err
		}
		if err := service.BootstrapAdmin(ctx, *name, *email, os.Getenv("ADMIN_PASSWORD")); err != nil {
			return err
		}
		fmt.Println("Administrator created.")
	case actionCreateCredential:
		password := os.Getenv("CLIENT_PASSWORD")
		if strings.TrimSpace(*username) == "" || strings.Contains(*username, ":") || len(*username) > security.MaximumUsernameLength || len(password) < security.MinimumPasswordLength || len(password) > security.MaximumPasswordLength {
			return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		hash, err := security.HashPassword(password)
		if err != nil {
			return apperrors.Wrap(apperrors.CodeInternal, err)
		}
		credential := models.APICredential{Username: *username, PasswordHash: hash, Active: true}
		if err := repositories.NewCredentials(db).Create(ctx, &credential); err != nil {
			return apperrors.Wrap(apperrors.CodeInvalidRequest, err)
		}
		fmt.Printf("Credential created: %s (id=%d)\n", credential.Username, credential.ID)
	default:
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		slog.Error("Administrative action failed", "error", err)
		os.Exit(config.ExitFailure)
	}
}
