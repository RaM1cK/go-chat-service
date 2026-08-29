package service

import (
	"context"
	"spotify-chat/internal/dto"
	"spotify-chat/internal/repository"
)

type msgSrv struct {
	msgRepo repository.MessageRepository
}

func NewMessageService(msgRepo repository.MessageRepository) MessageService {
	return &msgSrv{
		msgRepo: msgRepo,
	}
}

func (s *msgSrv) SaveMessage(ctx context.Context, msgParams dto.CreateMessageParams) (dto.Message, error) {
	return s.msgRepo.Create(ctx, msgParams)
}