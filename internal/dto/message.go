package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateMessageParams struct {
	ChatID   uuid.UUID `json:"chatId"`
	SenderID uuid.UUID `json:"senderId"`
	Data     string    `json:"data"`
	DataType int16     `json:"dataType"`
	QuotedID *int64    `json:"quotedId"`
}

type Message struct {
	ChatID    uuid.UUID `json:"chatId"`
	CreatedAt time.Time `json:"createdAt"`
	ID        int64     `json:"id"`
	SenderID  uuid.UUID `json:"senderId"`
	Data      string    `json:"data"`
	DataType  int16     `json:"dataType"`
	QuotedID  *int64    `json:"quotedId"`
}
