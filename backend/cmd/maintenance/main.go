package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/database"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/services"
	"log/slog"
	"os"
	"time"
)

type maintenanceAction string

const (
	actionRotate  maintenanceAction = "rotate-keys"
	actionPurgeMO maintenanceAction = "purge-mo"
)

func run() error {
	action := flag.String("action", "", "rotate-keys or purge-mo")
	apply := flag.Bool("apply", false, "Apply writes; omitted means dry-run")
	flag.Parse()
	if maintenanceAction(*action) != actionRotate && maintenanceAction(*action) != actionPurgeMO {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	pool, _ := db.DB()
	defer pool.Close()
	tokens, err := cfg.TokenVault()
	if err != nil {
		return err
	}
	mo, err := cfg.MOVault()
	if err != nil {
		return err
	}
	service := services.NewMaintenanceService(repositories.NewMaintenance(db), tokens, mo)
	if maintenanceAction(*action) == actionRotate {
		result, err := service.Rotate(ctx, *apply)
		encodeErr := json.NewEncoder(os.Stdout).Encode(map[string]any{"apply": *apply, "completed": err == nil, "result": result})
		if err != nil {
			return err
		}
		return encodeErr
	}
	count, err := service.PurgeMO(ctx, cfg.MORetentionDays, *apply)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Apply bool  `json:"apply"`
		Count int64 `json:"batchCount"`
	}{*apply, count})
}
func main() {
	if err := run(); err != nil {
		slog.Error("Maintenance failed")
		os.Exit(config.ExitFailure)
	}
}
