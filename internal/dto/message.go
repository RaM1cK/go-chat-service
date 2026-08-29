package dto

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ChatID    uuid.UUID
	CreatedAt time.Time
	ID        int64
	SenderID  uuid.UUID
	Data      string
	DataType  int16
	QuotedID  *int64
}

type CreateMessageParams struct {
	ChatID   uuid.UUID
	SenderID uuid.UUID
	Data     string
	DataType int16
	QuotedID *int64
}
