package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateMessageParams struct {
	ID       uuid.UUID  `json:"id"`
	ChatID   uuid.UUID  `json:"chatId"`
	SenderID uuid.UUID  `json:"senderId"`
	Data     string     `json:"data"`
	DataType int16      `json:"dataType"`
	QuotedID *uuid.UUID `json:"quotedId"`
}

type Message struct {
	ChatID    uuid.UUID  `json:"chatId"`
	CreatedAt time.Time  `json:"createdAt"`
	ID        uuid.UUID  `json:"id"`
	SenderID  uuid.UUID  `json:"senderId"`
	Data      string     `json:"data"`
	DataType  int16      `json:"dataType"`
	QuotedID  *uuid.UUID `json:"quotedId"`
}
