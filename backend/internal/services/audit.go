package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type AuditService struct{ repository *repositories.Audits }

func NewAuditService(repo *repositories.Audits) *AuditService { return &AuditService{repository: repo} }
func (s *AuditService) Audits(ctx context.Context, q dtos.AuditListQuery) (dtos.Page[dtos.AuditView], error) {
	values, err := s.repository.Audits(ctx, q)
	return mapPage(values, func(v models.AuditEvent) dtos.AuditView {
		return dtos.AuditView{ID: v.ID, ActorID: v.ActorID, ActorName: v.ActorName, Action: v.Action, Resource: v.Resource, ResourceID: v.ResourceID, CreatedAt: v.CreatedAt}
	}), dbError(err)
}
