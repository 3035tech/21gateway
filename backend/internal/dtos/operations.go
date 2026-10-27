package dtos

import "time"

type QueueDiagnostics struct {
	UnmatchedEvents   int64      `json:"unmatchedEvents"`
	OldestUnmatchedAt *time.Time `json:"oldestUnmatchedAt"`
	Pending           int64      `json:"pending"`
	Processing        int64      `json:"processing"`
	ExpiredLeases     int64      `json:"expiredLeases"`
	FailedLastHour    int64      `json:"failedLastHour"`
	OldestPendingAt   *time.Time `json:"oldestPendingAt"`
}
type ScheduledQueueDiagnostics struct {
	Ready               int64      `json:"ready"`
	OpenBatches         int64      `json:"openBatches"`
	UnpublishedBatches  int64      `json:"unpublishedBatches"`
	OldestOpenBatchAt   *time.Time `json:"oldestOpenBatchAt"`
	OldestUnpublishedAt *time.Time `json:"oldestUnpublishedAt"`
	Queued              int64      `json:"queued"`
	Due                 int64      `json:"due"`
	Claimed             int64      `json:"claimed"`
	Dispatching         int64      `json:"dispatching"`
	UnknownLastHour     int64      `json:"unknownLastHour"`
	FailedLastHour      int64      `json:"failedLastHour"`
	OldestDueAt         *time.Time `json:"oldestDueAt"`
}
type OperationsSnapshot struct {
	Scheduler  ScheduledQueueDiagnostics `json:"scheduler"`
	ObservedAt time.Time                 `json:"observedAt"`
	DLR        QueueDiagnostics          `json:"dlr"`
	MO         QueueDiagnostics          `json:"mo"`
}
type OperationsOutput struct{ Body OperationsSnapshot }
