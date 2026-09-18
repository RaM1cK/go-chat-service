package service

import (
	"context"
	"spotify-chat/internal/dto"
	"spotify-chat/internal/repository"
	"time"

	"github.com/google/uuid"
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

func (s *msgSrv) GetMessages(ctx context.Context, chatID uuid.UUID, limit int32, timeOffset *time.Time) ([]dto.Message, error) {
	return s.msgRepo.GetByChatID(ctx, chatID, limit, timeOffset)
}

func (s *msgSrv) GetLastMessagesByChatIDs(ctx context.Context, chatIDs []uuid.UUID) (map[uuid.UUID]dto.Message, error) {
	return s.msgRepo.GetLastMessagesByChatIDs(ctx, chatIDs)
}
