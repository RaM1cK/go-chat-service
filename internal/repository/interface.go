package repository

import (
	"context"
	"spotify-chat/internal/dto"
	"time"

	"github.com/google/uuid"
)

type MessageRepository interface {
	Create(ctx context.Context, params dto.CreateMessageParams) (dto.Message, error)
	GetByChatID(ctx context.Context, chatID uuid.UUID, limit int) ([]dto.Message, error)
	Delete(ctx context.Context, chatID uuid.UUID, createdAt time.Time, id uuid.UUID) error
}

type ChatRepository interface {
	Create(ctx context.Context, params dto.CreateChatParams) (dto.Chat, error)
	GetChatsByUserId(ctx context.Context, userID uuid.UUID) ([]dto.Chat, error)
	AddUser(ctx context.Context, userID uuid.UUID, chatID uuid.UUID) error
	RemoveUser(ctx context.Context, userID uuid.UUID, chatID uuid.UUID) error
	Delete(ctx context.Context, chatID uuid.UUID) error
}
