package dtos

import (
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
)

type ScheduledListQuery struct {
	Page         int                  `query:"page" default:"1" minimum:"1"`
	PageSize     int                  `query:"pageSize" default:"25" minimum:"1" maximum:"100"`
	CompanyID    uint64               `query:"companyId"`
	CredentialID uint64               `query:"credentialId"`
	State        models.ScheduleState `query:"state" enum:"queued,claimed,dispatching,done,failed,unknown"`
	Search       string               `query:"search" maxLength:"200"`
	DueFrom      string               `query:"dueFrom" format:"date-time"`
	DueTo        string               `query:"dueTo" format:"date-time"`
	Sort         string               `query:"sort" enum:"id,attemptId,companyName,username,destination,correlationId,scheduledAt,state,submissionState,dispatchedAt,createdAt"`
	Direction    SortDirection        `query:"direction" default:"asc" enum:"asc,desc"`
}

type ScheduledView struct {
	ID              uint64                 `json:"id"`
	AttemptID       uint64                 `json:"attemptId"`
	CompanyID       *uint64                `json:"companyId"`
	CompanyName     string                 `json:"companyName"`
	CredentialID    uint64                 `json:"credentialId"`
	Username        string                 `json:"username"`
	Destination     string                 `json:"destination"`
	CorrelationID   string                 `json:"correlationId"`
	ScheduledAt     time.Time              `json:"scheduledAt"`
	DispatchedAt    *time.Time             `json:"dispatchedAt"`
	State           models.ScheduleState   `json:"state"`
	SubmissionState models.SubmissionState `json:"submissionState"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type SchedulesOutput struct{ Body Page[ScheduledView] }
