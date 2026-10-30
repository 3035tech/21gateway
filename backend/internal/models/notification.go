package models

import "time"

type NotificationType string

const (
	NotificationUnknownSends   NotificationType = "unknown_sends"
	NotificationExpiredLeases  NotificationType = "expired_leases"
	NotificationDLRPending     NotificationType = "dlr_pending"
	NotificationMOPending      NotificationType = "mo_pending"
	NotificationDLRUnmatched   NotificationType = "dlr_unmatched"
	NotificationMOUnmatched    NotificationType = "mo_unmatched"
	NotificationScheduleDue    NotificationType = "schedule_overdue"
	NotificationCallbackFailed NotificationType = "callback_failed"
)

type NotificationEvent struct {
	ID            uint64
	TypeCode      NotificationType
	Title         string
	Description   string
	Path          string
	EmailSelected bool
	OpenedAt      time.Time
	UpdatedAt     time.Time
	ResolvedAt    *time.Time
}

type NotificationEmail struct {
	EventID     uint64
	UserID      uint64
	To          string
	Title       string
	Description string
	Path        string
	Attempts    int
}
