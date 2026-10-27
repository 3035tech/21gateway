package dtos

import (
	"time"
)

type AuditView struct {
	ID         uint64    `json:"id"`
	ActorID    *uint64   `json:"actorId"`
	ActorName  string    `json:"actorName"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID *uint64   `json:"resourceId"`
	CreatedAt  time.Time `json:"createdAt"`
}
type AuditOutput struct{ Body Page[AuditView] }

type AuditListQuery struct {
	Page      int           `query:"page" default:"1" minimum:"1"`
	PageSize  int           `query:"pageSize" default:"25" minimum:"1" maximum:"100"`
	Search    string        `query:"search" maxLength:"200"`
	Sort      string        `query:"sort" enum:"id,actorName,action,resource,resourceId,createdAt"`
	Direction SortDirection `query:"direction" default:"asc" enum:"asc,desc"`
	ActorID   uint64        `query:"actorId"`
	Action    string        `query:"action" maxLength:"64"`
	Resource  string        `query:"resource" maxLength:"32"`
}
