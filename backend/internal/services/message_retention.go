package services

import (
	"context"
	"log/slog"
	"time"
)

const messageRetentionInterval = 5 * time.Minute
const messageRetentionBatchSize = 500
const messageRetentionBatchesPerCycle = 10

type MessageContentPurger interface {
	PurgeContent(context.Context, time.Time, int) (int64, error)
}

type MessageRetentionWorker struct{ repository MessageContentPurger }

func NewMessageRetentionWorker(repository MessageContentPurger) *MessageRetentionWorker {
	return &MessageRetentionWorker{repository: repository}
}

func (w *MessageRetentionWorker) Run(ctx context.Context) {
	cycle := func() {
		bounded, cancel := context.WithTimeout(ctx, workerCycleTimeout)
		defer cancel()
		before := time.Now().UTC().Add(-SMSHistoryRetentionDays * 24 * time.Hour)
		for range messageRetentionBatchesPerCycle {
			count, err := w.repository.PurgeContent(bounded, before, messageRetentionBatchSize)
			if err != nil {
				slog.Error("SMS content retention failed")
				return
			}
			if count > 0 {
				slog.Info("SMS content purged", "count", count)
			}
			if count < messageRetentionBatchSize {
				return
			}
		}
	}
	cycle()
	ticker := time.NewTicker(messageRetentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cycle()
		}
	}
}
