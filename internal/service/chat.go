package service

import (
	"context"
	"spotify-chat/internal/dto"
	"spotify-chat/internal/repository"

	"github.com/google/uuid"
)

type chatService struct {
	chatRepo repository.ChatRepository
}

func NewChatService(chatRepo repository.ChatRepository) ChatService {
	return &chatService{
		chatRepo: chatRepo,
	}
}

func (s *chatService) CreateChat(ctx context.Context, params dto.CreateChatParams) (dto.Chat, error) {
	chat, err := s.chatRepo.Create(ctx, params)
	if err != nil {
		return dto.Chat{}, err
	}

	return chat, nil
}

func (s *chatService) GetChatsByUserId(ctx context.Context, userId uuid.UUID) ([]dto.Chat, error) {
	chats, err := s.chatRepo.GetChatsByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	return chats, err
}