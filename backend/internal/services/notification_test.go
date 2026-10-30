package services

import (
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
)

func TestNotificationEventsUseOperationalThresholds(t *testing.T) {
	now := time.Now().UTC()
	before := func(age time.Duration) *time.Time { value := now.Add(-age); return &value }
	snapshot := dtos.OperationsSnapshot{ObservedAt: now, UnresolvedSends: 2}
	snapshot.DLR.Pending = 1
	snapshot.DLR.OldestPendingAt = before(14 * time.Minute)
	snapshot.MO.UnmatchedEvents = 3
	snapshot.MO.OldestUnmatchedAt = before(16 * time.Minute)
	snapshot.Scheduler.Due = 1
	snapshot.Scheduler.OldestDueAt = before(6 * time.Minute)
	snapshot.DLR.FailedLastHour = 1
	events := notificationEvents(snapshot)
	want := map[models.NotificationType]bool{
		models.NotificationUnknownSends: true,
		models.NotificationMOUnmatched: true,
		models.NotificationScheduleDue: true,
		models.NotificationCallbackFailed: true,
	}
	if len(events) != len(want) { t.Fatalf("events=%+v", events) }
	for _, event := range events {
		if !want[event.TypeCode] || event.Path == "" || event.Description == "" { t.Fatalf("unexpected event %+v", event) }
	}
}
