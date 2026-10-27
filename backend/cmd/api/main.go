package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/21mobile/sms-gateway/internal/background"
	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/database"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/httpapi"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/services"
)

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, config.StartupTimeout)
	db, err := database.Open(startup, cfg.DatabaseURL)
	cancel()
	if err != nil {
		return apperrors.Wrap(apperrors.CodeDatabaseUnavailable, err)
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	provider, err := whitelabel.New(cfg.WhiteLabelBaseURL, "")
	if err != nil {
		return err
	}
	defer provider.Close()
	auth, err := services.NewAuthServiceWithProtection(repositories.NewCredentials(db), repositories.NewAuthenticationAttempts(db), cfg.APIAuthenticationIPLimit)
	if err != nil {
		return err
	}
	users, err := services.NewUserService(repositories.NewUsers(db))
	if err != nil {
		return err
	}
	sessions := httpapi.NewSessions(pool, cfg.CookieSecure, cfg.CookieCrossSite)
	vault, err := cfg.TokenVault()
	if err != nil {
		return err
	}
	moVault, err := cfg.MOVault()
	if err != nil {
		return err
	}
	sender := services.NewSMSServiceWithLimiter(repositories.NewSMSAttempts(db), services.NewAccountSMSProvider(provider, vault), repositories.NewSendLimits(db))
	if cfg.SchedulerEnabled {
		scheduledVault, err := cfg.ScheduledVault()
		if err != nil {
			return err
		}
		sender.SetScheduler(services.NewScheduledSMSService(repositories.NewScheduledSMS(db), scheduledVault))
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.New(cfg.AllowedOrigins, cfg.DocsEnabled, pool.PingContext, httpapi.Dependencies{Operations: services.NewOperationsService(repositories.NewOperations(db)), OperationsEnabled: cfg.OperationsEnabled, OperationsToken: cfg.OperationsToken, LegacyMO: services.NewLegacyMOService(repositories.NewMOs(db), moVault), MO: services.NewMOService(repositories.NewMOs(db), moVault), MOIngressEnabled: cfg.MOIngressEnabled, LegacyStatuses: services.NewLegacyStatusService(repositories.NewLegacyStatuses(db)), SMS: sender, Auth: auth, Users: users, Companies: services.NewCompanyService(repositories.NewCompanies(db)), Credentials: services.NewCredentialServiceWithVault(repositories.NewCredentials(db), vault), Messages: services.NewMessageService(repositories.NewSMSAttempts(db)), Schedules: services.NewScheduledListService(repositories.NewScheduledSMS(db)), Audit: services.NewAuditService(repositories.NewAudits(db)), Summary: services.NewSummaryService(repositories.NewSummary(db)), Sessions: sessions, Callbacks: services.NewCallbackService(repositories.NewCallbacks(db)), CallbackIngressEnabled: cfg.CallbackIngressEnabled, CallbackIngressToken: cfg.CallbackIngressToken}),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}
	workers, err := background.New(ctx, cfg, db, provider, vault, moVault)
	if err != nil {
		return err
	}
	workerCtx, stopWorkers := context.WithCancel(ctx)
	workersDone := make(chan struct{})
	go func() {
		defer close(workersDone)
		workers.Run(workerCtx)
	}()
	defer func() {
		stopWorkers()
		<-workersDone
		workers.Close()
	}()
	failures := make(chan error, 1)
	go func() { slog.Info("API started", "address", cfg.Address); failures <- server.ListenAndServe() }()
	select {
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
	}
	return nil
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(config.ExitFailure)
	}
}
