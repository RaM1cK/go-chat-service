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
	Delete(ctx context.Context, chatID uuid.UUID, createdAt time.Time, id int64) error
}
