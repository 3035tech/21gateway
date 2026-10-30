package dtos

import "time"

type NotificationItem struct {
	ID          uint64     `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Path        string     `json:"path"`
	OpenedAt    time.Time  `json:"openedAt"`
	ResolvedAt  *time.Time `json:"resolvedAt"`
	ReadAt      *time.Time `json:"readAt"`
}

type NotificationList struct {
	Items       []NotificationItem `json:"items"`
	UnreadCount int64              `json:"unreadCount"`
	Total       int64              `json:"total"`
	Page        int                `json:"page"`
	PageSize    int                `json:"pageSize"`
}

type NotificationListInput struct {
	Page     int    `query:"page" default:"1" minimum:"1"`
	PageSize int    `query:"pageSize" default:"20" minimum:"1" maximum:"100"`
	Unread   bool   `query:"unread"`
	Sort     string `query:"sort" default:"openedAt" enum:"openedAt,title"`
	Order    string `query:"order" default:"desc" enum:"asc,desc"`
}
type NotificationListOutput struct{ Body NotificationList }

type NotificationReadInput struct {
	ID uint64 `path:"id" minimum:"1"`
}
type NotificationTypeInput struct {
	Code string `path:"code"`
}
type NotificationPreferenceBody struct {
	EmailEnabled bool `json:"emailEnabled"`
}
type NotificationPreferenceInput struct {
	Code string `path:"code"`
	Body NotificationPreferenceBody
}
type NotificationPreference struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	EmailEnabled bool   `json:"emailEnabled"`
}
type NotificationPreferencesOutput struct {
	Body struct {
		Items          []NotificationPreference `json:"items"`
		EmailAvailable bool                     `json:"emailAvailable"`
	}
}
type NotificationPreferenceOutput struct{ Body NotificationPreference }
