package service

import (
	"context"
	"spotify-chat/internal/dto"
)

type MessageService interface {
	SaveMessage(ctx context.Context, msg dto.CreateMessageParams) (dto.Message, error)
}