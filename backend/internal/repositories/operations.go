package repositories

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"gorm.io/gorm"
	"time"
)

type Operations struct{ db *gorm.DB }

func NewOperations(db *gorm.DB) *Operations { return &Operations{db: db} }
func (r *Operations) Snapshot(ctx context.Context) (dtos.OperationsSnapshot, error) {
	snapshot := dtos.OperationsSnapshot{ObservedAt: time.Now().UTC()}
	const activeSQL = `SELECT COUNT(*) FILTER (WHERE state='pending') AS pending, COUNT(*) FILTER (WHERE state='processing') AS processing, COUNT(*) FILTER (WHERE state='processing' AND lease_until < NOW()) AS expired_leases, MIN(created_at) FILTER (WHERE state='pending') AS oldest_pending_at FROM callback_jobs WHERE state IN ('pending','processing')`
	const moActiveSQL = `SELECT COUNT(*) FILTER (WHERE state='pending') AS pending, COUNT(*) FILTER (WHERE state='processing') AS processing, COUNT(*) FILTER (WHERE state='processing' AND lease_until < NOW()) AS expired_leases, MIN(created_at) FILTER (WHERE state='pending') AS oldest_pending_at FROM mo_callback_jobs WHERE state IN ('pending','processing')`
	db := r.db.WithContext(ctx)
	if err := db.Raw(activeSQL).Scan(&snapshot.DLR).Error; err != nil {
		return snapshot, err
	}
	if err := db.Raw(moActiveSQL).Scan(&snapshot.MO).Error; err != nil {
		return snapshot, err
	}
	if err := db.Table("callback_jobs").Where("state='failed' AND updated_at >= ?", snapshot.ObservedAt.Add(-time.Hour)).Count(&snapshot.DLR.FailedLastHour).Error; err != nil {
		return snapshot, err
	}
	if err := db.Table("mo_callback_jobs").Where("state='failed' AND updated_at >= ?", snapshot.ObservedAt.Add(-time.Hour)).Count(&snapshot.MO.FailedLastHour).Error; err != nil {
		return snapshot, err
	}
	type unmatched struct {
		Count  int64
		Oldest *time.Time
	}
	var dlrEvents, moEvents unmatched
	if err := db.Raw("SELECT COUNT(*) AS count, MIN(created_at) AS oldest FROM delivery_events WHERE NOT processed").Scan(&dlrEvents).Error; err != nil {
		return snapshot, err
	}
	if err := db.Raw("SELECT COUNT(*) AS count, MIN(created_at) AS oldest FROM mo_events WHERE NOT processed").Scan(&moEvents).Error; err != nil {
		return snapshot, err
	}
	snapshot.DLR.UnmatchedEvents = dlrEvents.Count
	snapshot.DLR.OldestUnmatchedAt = dlrEvents.Oldest
	snapshot.MO.UnmatchedEvents = moEvents.Count
	snapshot.MO.OldestUnmatchedAt = moEvents.Oldest
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state='queued') AS queued,COUNT(*) FILTER (WHERE state='queued' AND due_at<=NOW() AND available_at<=NOW()) AS due,COUNT(*) FILTER (WHERE state='claimed' AND lease_until IS NOT NULL) AS claimed,COUNT(*) FILTER (WHERE state='claimed' AND dispatch_batch_id IS NOT NULL AND lease_until IS NULL) AS ready,COUNT(*) FILTER (WHERE state='dispatching') AS dispatching,MIN(due_at) FILTER (WHERE state='queued' AND due_at<=NOW()) AS oldest_due_at FROM scheduled_sms WHERE state IN ('queued','claimed','dispatching')`).Scan(&snapshot.Scheduler).Error; err != nil {
		return snapshot, err
	}
	var recent struct {
		UnknownLastHour int64
		FailedLastHour  int64
	}
	if err := db.Raw(`SELECT COUNT(*) FILTER (WHERE state='unknown') AS unknown_last_hour,COUNT(*) FILTER (WHERE state='failed') AS failed_last_hour FROM scheduled_sms WHERE state IN ('unknown','failed') AND updated_at>=?`, snapshot.ObservedAt.Add(-time.Hour)).Scan(&recent).Error; err != nil {
		return snapshot, err
	}
	snapshot.Scheduler.UnknownLastHour = recent.UnknownLastHour
	snapshot.Scheduler.FailedLastHour = recent.FailedLastHour
	var outbox struct {
		OpenBatches         int64
		UnpublishedBatches  int64
		OldestOpenBatchAt   *time.Time
		OldestUnpublishedAt *time.Time
	}
	if err := db.Raw(`SELECT COUNT(*) AS open_batches,COUNT(*) FILTER (WHERE published_at IS NULL) AS unpublished_batches,MIN(created_at) AS oldest_open_batch_at,MIN(created_at) FILTER (WHERE published_at IS NULL) AS oldest_unpublished_at FROM sms_dispatch_batches WHERE completed_at IS NULL`).Scan(&outbox).Error; err != nil {
		return snapshot, err
	}
	snapshot.Scheduler.OpenBatches = outbox.OpenBatches
	snapshot.Scheduler.UnpublishedBatches = outbox.UnpublishedBatches
	snapshot.Scheduler.OldestOpenBatchAt = outbox.OldestOpenBatchAt
	snapshot.Scheduler.OldestUnpublishedAt = outbox.OldestUnpublishedAt
	return snapshot, nil
}
