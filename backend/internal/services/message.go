package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type MessageService struct{ repository *repositories.SMSAttempts }

func NewMessageService(repo *repositories.SMSAttempts) *MessageService {
	return &MessageService{repository: repo}
}
func (s *MessageService) Messages(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.MessageView], error) {
	values, err := s.repository.Messages(ctx, q)
	return values, dbError(err)
}
