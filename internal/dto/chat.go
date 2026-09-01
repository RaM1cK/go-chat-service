package dto

import "github.com/google/uuid"

type CreateChatParams struct {
	Type int16
	Name *string
	Logo *string
}

type Chat struct {
	ID   uuid.UUID
	Type int16
	Name *string
	Logo *string
}

