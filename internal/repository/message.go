package repository

import (
	"context"
	"spotify-chat/internal/db/scylla"
	"spotify-chat/internal/dto"
	"time"

	"github.com/google/uuid"
	"github.com/scylladb/gocqlx/v3"
	"github.com/scylladb/gocqlx/v3/qb"
)

type messageRepo struct {
	session gocqlx.Session
}

func NewMessageRepo(session gocqlx.Session) MessageRepository {
	return &messageRepo{
		session: session,
	}
}

func (r *messageRepo) Create(ctx context.Context, params dto.CreateMessageParams) (dto.Message, error) {
	now := time.Now()

	row := scylla.MessagesByChatStruct{
		ChatId:    params.ChatID,
		CreatedAt: now,
		Data:      params.Data,
		DataType:  params.DataType,
		Id:        uuid.UUID(params.ID),
		SenderId:  params.SenderID,
	}

	if params.QuotedID != nil {
		row.QuotedId = uuid.UUID(*params.QuotedID)
	}

	if err := scylla.MessagesByChat.
		InsertQueryContext(ctx, r.session).
		BindStruct(row).
		ExecRelease(); err != nil {
		return dto.Message{}, err
	}

	return toMessage(row), nil
}

func (r *messageRepo) GetByChatID(ctx context.Context, chatID uuid.UUID, limit int) ([]dto.Message, error) {
	var rows []scylla.MessagesByChatStruct

	q := qb.Select("messages_by_chat").
		Where(qb.Eq("chat_id")).
		LimitNamed("limit").
		QueryContext(ctx, r.session).
		BindStructMap(scylla.MessagesByChatStruct{
			ChatId: chatID,
		}, map[string]interface{}{
			"limit": limit,
		})

	if err := q.SelectRelease(&rows); err != nil {
		return nil, err
	}

	return toMessages(rows), nil
}

func (r *messageRepo) Delete(ctx context.Context, chatID uuid.UUID, createdAt time.Time, id uuid.UUID) error {
	q := scylla.MessagesByChat.
		DeleteQueryContext(ctx, r.session).
		BindStruct(scylla.MessagesByChatStruct{
			ChatId:    chatID,
			CreatedAt: createdAt,
			Id:        uuid.UUID(id),
		})

	return q.ExecRelease()
}

func toMessages(rows []scylla.MessagesByChatStruct) []dto.Message {
	msgs := make([]dto.Message, len(rows))
	for i, row := range rows {
		msgs[i] = toMessage(row)
	}
	return msgs
}

func toMessage(m scylla.MessagesByChatStruct) dto.Message {
	msg := dto.Message{
		CreatedAt: m.CreatedAt,
		ID:        uuid.UUID(m.Id),
		ChatID:    m.ChatId,
		SenderID:  m.SenderId,
		Data:      m.Data,
		DataType:  m.DataType,
	}

	if m.QuotedId != uuid.Nil {
		q := uuid.UUID(m.QuotedId)
		msg.QuotedID = &q
	}

	return msg
}
