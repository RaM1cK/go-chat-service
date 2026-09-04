package repository

import (
	"context"
	"spotify-chat/internal/db/pgsql"
	"spotify-chat/internal/dto"

	"github.com/google/uuid"
)

type chatRepo struct {
	queries *pgsql.Queries
}

func NewChatRepo(queries *pgsql.Queries) ChatRepository {
	return &chatRepo{
		queries: queries,
	}
}

func (r *chatRepo) Create(ctx context.Context, params dto.CreateChatParams) (dto.Chat, error) {
	chat, err := r.queries.CreateChat(ctx, pgsql.CreateChatParams(params))

	if err != nil {
		return dto.Chat{}, err
	}

	return dto.Chat(chat), nil
}

func (r *chatRepo) GetChatsByUserId(ctx context.Context, userID uuid.UUID) ([]dto.Chat, error) {
	chats, err := r.queries.GetChatsByUserId(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.Chat, len(chats))
	for i, chat := range chats {
		result[i] = dto.Chat(chat)
	}

	return result, err
}
	
func (r *chatRepo) AddUser(ctx context.Context, userID uuid.UUID, chatID uuid.UUID) error {
	return r.queries.AddUser(ctx, pgsql.AddUserParams{
		UserID: userID,
		ChatID: chatID,
	})
}
	
func (r *chatRepo) RemoveUser(ctx context.Context, userID uuid.UUID, chatID uuid.UUID) error {
	return r.queries.RemoveUser(ctx, pgsql.RemoveUserParams{
		UserID: userID,
		ChatID: chatID,
	})
}
	
func (r *chatRepo) Delete(ctx context.Context, chatID uuid.UUID) error {
	return r.queries.DeleteChat(ctx, chatID)
}