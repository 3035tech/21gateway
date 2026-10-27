package dtos

type Summary struct {
	Companies          int64 `json:"companies"`
	ActiveCredentials  int64 `json:"activeCredentials"`
	Accepted           int64 `json:"accepted"`
	Unknown            int64 `json:"unknown"`
	Rejected           int64 `json:"rejected"`
	PendingCallbacks   int64 `json:"pendingCallbacks"`
	FailedCallbacks    int64 `json:"failedCallbacks"`
	PendingMOCallbacks int64 `json:"pendingMoCallbacks"`
	FailedMOCallbacks  int64 `json:"failedMoCallbacks"`
	ScheduledQueued    int64 `json:"scheduledQueued"`
	ScheduledDue       int64 `json:"scheduledDue"`
	ScheduledNext24h   int64 `json:"scheduledNext24h"`
	ScheduledActive    int64 `json:"scheduledActive"`
}
type SummaryOutput struct{ Body Summary }
