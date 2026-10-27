package services

import (
	"context"
	"log/slog"
	"time"
)

const retentionInterval = 5 * time.Minute
const retentionBatchesPerCycle = 10

func (s *MaintenanceService) RunRetention(ctx context.Context, days int) {
	if days <= 0 {
		return
	}
	cycle := func() {
		bounded, cancel := context.WithTimeout(ctx, workerCycleTimeout)
		defer cancel()
		for range retentionBatchesPerCycle {
			count, err := s.PurgeMO(bounded, days, true)
			if err != nil {
				slog.Error("MO retention failed")
				return
			}
			if count > 0 {
				slog.Info("MO content purged", "count", count)
			}
			if count < MaintenanceBatchSize {
				return
			}
		}
	}
	cycle()
	ticker := time.NewTicker(retentionInterval)
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
