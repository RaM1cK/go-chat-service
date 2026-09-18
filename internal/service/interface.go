package service

import (
	"context"
	"spotify-chat/internal/dto"
	"time"

	"github.com/google/uuid"
)

type MessageService interface {
	SaveMessage(ctx context.Context, params dto.CreateMessageParams) (dto.Message, error)
	GetMessages(ctx context.Context, chatID uuid.UUID, limit int32, timeOffset *time.Time) ([]dto.Message, error)
	GetLastMessagesByChatIDs(ctx context.Context, chatIDs []uuid.UUID) (map[uuid.UUID]dto.Message, error)
}

type ChatService interface {
	CreateChat(ctx context.Context, params dto.CreateChatParams) (dto.Chat, error)
	GetChatsByUserId(ctx context.Context, userId uuid.UUID) ([]dto.Chat, error)
}
