package service

import (
	"context"
	"spotify-chat/internal/dto"
)

type MessageService interface {
	SaveMessage(ctx context.Context, params dto.CreateMessageParams) (dto.Message, error)
}

type ChatService interface {
	CreateChat(ctx context.Context, params dto.CreateChatParams) (dto.Chat, error)
}