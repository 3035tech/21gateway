// Package background composes the gateway's existing domain workers.
package background

import (
	"context"
	"log/slog"
	"sync"

	"github.com/21mobile/sms-gateway/internal/config"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/callback"
	"github.com/21mobile/sms-gateway/internal/integrations/dispatchqueue"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"gorm.io/gorm"
)

type Workers struct {
	callback         *services.CallbackWorker
	mo               *services.MOWorker
	retention        *services.MaintenanceService
	messageRetention *services.MessageRetentionWorker
	scheduled        *services.ScheduledSMSWorker
	pipeline         *services.DispatchPipeline
	sendReferences   *services.SendReferencePipeline
	sender           *callback.Client
	queue            *dispatchqueue.Client
	days             int
}

func New(ctx context.Context, cfg config.Config, db *gorm.DB, provider *whitelabel.Client, tokenVault *security.TokenVault, moVault *security.MOContentVault) (*Workers, error) {
	w := &Workers{sender: callback.New(), days: cfg.MORetentionDays}
	if cfg.SMSSQSEnabled {
		var open int64
		if err := db.WithContext(ctx).Model(&models.DispatchBatch{}).Where("completed_at IS NULL").Count(&open).Error; err != nil {
			w.Close()
			return nil, err
		}
		if open != 0 {
			w.Close()
			return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
	}
	w.messageRetention = services.NewMessageRetentionWorker(repositories.NewSMSAttempts(db))
	w.callback = services.NewCallbackWorker(repositories.NewCallbacks(db), w.sender)
	if moVault.Enabled() {
		w.mo = services.NewMOWorker(repositories.NewMOs(db), w.sender, moVault)
	}
	if cfg.MORetentionDays > 0 {
		w.retention = services.NewMaintenanceService(repositories.NewMaintenance(db), nil, nil)
	}
	if cfg.SchedulerEnabled {
		scheduledVault, err := cfg.ScheduledVault()
		if err != nil {
			w.Close()
			return nil, err
		}
		w.scheduled = services.NewScheduledSMSWorker(repositories.NewScheduledSMS(db), services.NewAccountSMSProvider(provider, tokenVault), repositories.NewDispatchSendLimits(db), scheduledVault)
		if cfg.ScheduledSQSEnabled || cfg.SMSSQSEnabled {
			startup, cancel := context.WithTimeout(ctx, config.StartupTimeout)
			queueURL := cfg.ScheduledSQSQueueURL
			if cfg.SMSSQSEnabled {
				queueURL = cfg.SMSSQSQueueURL
			}
			w.queue, err = dispatchqueue.New(startup, cfg.AWSRegion, queueURL)
			cancel()
			if err != nil {
				w.Close()
				return nil, err
			}
			if cfg.SMSSQSEnabled {
				w.sendReferences = services.NewSendReferencePipeline(repositories.NewSendReferences(db), w.queue, w.scheduled)
			} else {
				w.pipeline = services.NewDispatchPipeline(repositories.NewDispatchBatches(db), w.queue, w.scheduled)
			}
		}
	}
	return w, nil
}

// Run returns after the context is cancelled and all worker loops have stopped.
func (w *Workers) Run(ctx context.Context) {
	var group sync.WaitGroup
	start := func(run func(context.Context)) {
		group.Add(1)
		go func() { defer group.Done(); run(ctx) }()
	}
	if w.sendReferences != nil {
		start(w.sendReferences.Run)
	} else if w.pipeline != nil {
		start(w.pipeline.Run)
	} else if w.scheduled != nil {
		start(w.scheduled.Run)
	}
	if w.mo != nil {
		start(w.mo.Run)
	}
	if w.retention != nil {
		start(func(ctx context.Context) { w.retention.RunRetention(ctx, w.days) })
	}
	if w.messageRetention != nil {
		start(w.messageRetention.Run)
	}
	slog.Info("Callback worker started")
	w.callback.Run(ctx)
	group.Wait()
}

// Close must be called after Run has returned.
func (w *Workers) Close() {
	if w.queue != nil {
		w.queue.Close()
	}
	if w.sender != nil {
		w.sender.Close()
	}
}

func (w *Workers) QueueProbe() func(context.Context) error {
	if w.queue == nil {
		return nil
	}
	return w.queue.Probe
}
