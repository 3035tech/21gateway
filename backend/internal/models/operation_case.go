package models

import "time"

const (
	OperationCaseNoteMaxChars = 1000
	OperationCaseMaxPageSize  = 100
	OperationScheduleGrace    = 5 * time.Minute
)

type OperationCaseKind string

const (
	OperationUnknownSend     OperationCaseKind = "unknown_send"
	OperationOverdueSchedule OperationCaseKind = "overdue_schedule"
	OperationDLRCallback     OperationCaseKind = "dlr_callback"
	OperationMOCallback      OperationCaseKind = "mo_callback"
)

type OperationCaseStatus string

const (
	OperationCaseOpen       OperationCaseStatus = "open"
	OperationCaseInProgress OperationCaseStatus = "in_progress"
	OperationCaseCompleted  OperationCaseStatus = "completed"
)

func (s OperationCaseStatus) Valid() bool {
	return s == OperationCaseOpen || s == OperationCaseInProgress || s == OperationCaseCompleted
}
