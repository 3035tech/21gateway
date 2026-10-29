package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/21mobile/sms-gateway/internal/background"
	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/database"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/httpapi"
	"github.com/21mobile/sms-gateway/internal/integrations/mail"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
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
	provider.SetLogUnexpectedResponseBody(cfg.LogUnexpectedProviderResponse)
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
	mfaVault, err := cfg.MFAVault()
	if err != nil {
		return err
	}
	mfa := services.NewUserMFAService(repositories.NewUserMFA(db), repositories.NewUsers(db), mfaVault)
	users.SetMFA(mfa)
	var recoveryMailer services.PasswordResetMailer
	var smtpProbe func(context.Context) error
	if cfg.RecoverySMTPHost != "" {
		appURL, err := url.Parse(cfg.PublicAppURL)
		if err != nil {
			return err
		}
		smtpClient := &mail.SMTP{Host: cfg.RecoverySMTPHost, Port: cfg.RecoverySMTPPort, Username: cfg.RecoverySMTPUser, Password: cfg.RecoverySMTPPassword, From: cfg.RecoverySMTPFrom, AppURL: appURL}
		recoveryMailer = smtpClient
		smtpProbe = smtpClient.Probe
	}
	recovery := services.NewUserRecoveryService(repositories.NewUserRecovery(db), repositories.NewUsers(db), mfa, recoveryMailer)
	recovery.SetUserService(users)
	vault, err := cfg.TokenVault()
	if err != nil {
		return err
	}
	moVault, err := cfg.MOVault()
	if err != nil {
		return err
	}
	historyVault, err := cfg.SMSHistoryVault()
	if err != nil {
		return err
	}
	if !historyVault.Enabled() {
		return apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	sender := services.NewSMSServiceWithLimiter(repositories.NewSMSAttempts(db), services.NewAccountSMSProvider(provider, vault), repositories.NewSendLimits(db))
	if cfg.ZougCallbackEnabled {
		sender.SetCallbackLevel(whitelabel.CallbackAll)
	}
	sender.SetHistoryVault(historyVault)
	if cfg.SchedulerEnabled {
		scheduledVault, err := cfg.ScheduledVault()
		if err != nil {
			return err
		}
		scheduler := services.NewScheduledSMSService(repositories.NewScheduledSMS(db), scheduledVault)
		scheduler.SetHistoryVault(historyVault)
		if cfg.SMSSQSEnabled {
			sender.SetDispatchQueue(scheduler)
		} else {
			sender.SetScheduler(scheduler)
		}
	}
	workers, err := background.New(ctx, cfg, db, provider, vault, moVault)
	if err != nil {
		return err
	}
	defer workers.Close()
	clientIPs, err := security.NewClientIPResolver(cfg.TrustedProxyCIDRs)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeConfigurationInvalid, err)
	}
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.New(cfg.AllowedOrigins, cfg.DocsEnabled, pool.PingContext, httpapi.Dependencies{ClientIPResolver: clientIPs, SMTPProbe: smtpProbe, SQSProbe: workers.QueueProbe(), Operations: services.NewOperationsService(repositories.NewOperations(db)), OperationsEnabled: cfg.OperationsEnabled, OperationsToken: cfg.OperationsToken, HealthDependenciesTokenSHA256: cfg.HealthDependenciesTokenSHA256, LegacyMO: services.NewLegacyMOService(repositories.NewMOs(db), moVault), MO: services.NewMOService(repositories.NewMOs(db), moVault), MOIngressEnabled: cfg.MOIngressEnabled, ZougCallbackEnabled: cfg.ZougCallbackEnabled, ZougCallbackTokenSHA256: cfg.ZougCallbackTokenSHA256, ZougCallbackAllowedIPs: cfg.ZougCallbackAllowedIPs, LegacyStatuses: services.NewLegacyStatusService(repositories.NewLegacyStatuses(db)), SMS: sender, Submissions: services.NewSubmissionService(repositories.NewSMSAttempts(db)), Auth: auth, Users: users, UserMFA: mfa, UserRecovery: recovery, Companies: services.NewCompanyServiceWithVault(repositories.NewCompanies(db), vault), Credentials: services.NewCredentialServiceWithVault(repositories.NewCredentials(db), vault), Messages: services.NewMessageServiceWithTestSend(repositories.NewSMSAttempts(db), repositories.NewCredentials(db), repositories.NewAudits(db), sender, cfg.TestSMSDestination, cfg.TestSMSAllowAnyDestination).WithHistoryVault(historyVault).WithCompanyTests(repositories.NewCompanies(db)).WithDiagnosticDestinationHash(cfg.ZougDiagnosticDestinationSHA256), Schedules: services.NewScheduledListService(repositories.NewScheduledSMS(db)), Audit: services.NewAuditService(repositories.NewAudits(db)), Summary: services.NewSummaryService(repositories.NewSummary(db)), Sessions: sessions, Callbacks: services.NewCallbackService(repositories.NewCallbacks(db)), CallbackIngressEnabled: cfg.CallbackIngressEnabled, CallbackIngressToken: cfg.CallbackIngressToken}),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}
	if cfg.WorkersInAPI {
		workerCtx, stopWorkers := context.WithCancel(ctx)
		workersDone := make(chan struct{})
		go func() {
			defer close(workersDone)
			workers.Run(workerCtx)
		}()
		defer func() {
			stopWorkers()
			<-workersDone
		}()
	}
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
