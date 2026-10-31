package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type OperationCaseInput struct {
	CompanyID uint64                     `query:"companyId"`
	Search    string                     `query:"search" maxLength:"200"`
	Kind      models.OperationCaseKind   `query:"kind" enum:"unknown_send,overdue_schedule,dlr_callback,mo_callback"`
	Status    models.OperationCaseStatus `query:"status" enum:"open,in_progress,completed"`
	Sort      string                     `query:"sort" enum:"createdAt,companyName,kind,status,sourceId"`
	Direction SortDirection              `query:"direction" default:"desc" enum:"asc,desc"`
	Page      int                        `query:"page" default:"1" minimum:"1"`
	PageSize  int                        `query:"pageSize" default:"25" minimum:"1" maximum:"100"`
}

type OperationCaseItem struct {
	Kind         models.OperationCaseKind   `json:"kind"`
	SourceID     uint64                     `json:"sourceId"`
	AttemptID    uint64                     `json:"attemptId"`
	CompanyID    uint64                     `json:"companyId"`
	CompanyName  string                     `json:"companyName"`
	CreatedAt    time.Time                  `json:"createdAt"`
	Status       models.OperationCaseStatus `json:"status"`
	OwnerID      *uint64                    `json:"ownerId"`
	OwnerName    string                     `json:"ownerName"`
	Note         string                     `json:"note"`
	Tracked      bool                       `json:"tracked"`
	SourceActive bool                       `json:"sourceActive"`
}
type OperationCasesOutput struct{ Body Page[OperationCaseItem] }

type OperationCaseUpdateInput struct {
	Kind     models.OperationCaseKind `path:"kind"`
	SourceID uint64                   `path:"id" minimum:"1"`
	Body     struct {
		Status models.OperationCaseStatus `json:"status" enum:"open,in_progress,completed"`
		Note   string                     `json:"note" maxLength:"1000"`
	}
}
type OperationCaseEvent struct {
	ActorName string                     `json:"actorName"`
	Status    models.OperationCaseStatus `json:"status"`
	Note      string                     `json:"note"`
	CreatedAt time.Time                  `json:"createdAt"`
}
type OperationCaseHistoryInput struct {
	Kind     models.OperationCaseKind `path:"kind"`
	SourceID uint64                   `path:"id" minimum:"1"`
}
type OperationCaseHistoryOutput struct{ Body []OperationCaseEvent }
