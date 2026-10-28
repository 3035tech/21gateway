package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/21mobile/sms-gateway/internal/background"
	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/database"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
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
		return err
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
	tokens, err := cfg.TokenVault()
	if err != nil {
		return err
	}
	mo, err := cfg.MOVault()
	if err != nil {
		return err
	}
	workers, err := background.New(ctx, cfg, db, provider, tokens, mo)
	if err != nil {
		return err
	}
	defer workers.Close()
	workers.Run(ctx)
	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("Worker stopped", "error", err)
		os.Exit(config.ExitFailure)
	}
}
