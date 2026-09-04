package service

import (
	"context"
	"spotify-chat/internal/dto"

	"github.com/google/uuid"
)

type MessageService interface {
	SaveMessage(ctx context.Context, params dto.CreateMessageParams) (dto.Message, error)
}

type ChatService interface {
	CreateChat(ctx context.Context, params dto.CreateChatParams) (dto.Chat, error)
	GetChatsByUserId(ctx context.Context, userId uuid.UUID) ([]dto.Chat, error)
}
