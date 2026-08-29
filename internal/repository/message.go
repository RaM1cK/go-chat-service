package repository

import (
	"context"
	"spotify-chat/internal/dto"
	"spotify-chat/internal/db/scylla"
	"time"

	"github.com/google/uuid"
	"github.com/scylladb/gocqlx/v3"
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
	row := scylla.MessagesByChatStruct{
		ChatId:    [16]byte(params.ChatID),
    	CreatedAt: time.Now(),
    	Data:      params.Data,
    	DataType:  params.DataType,
    	Id:        time.Now().Unix(),
    	SenderId:  [16]byte(params.SenderID),
	}

	if params.QuotedID != nil {
		row.QuotedId = *params.QuotedID
	}

	q := scylla.MessagesByChat.
		InsertQueryContext(ctx, r.session).
		BindStruct(row)

	if err := q.ExecRelease(); err != nil {
		return dto.Message{}, err
	}

	return toMessage(row), nil
}

func (r *messageRepo) GetByChatID(ctx context.Context, chatID uuid.UUID, limit int) ([]dto.Message, error) {
	var rows []scylla.MessagesByChatStruct
	
	q := scylla.MessagesByChat.
		SelectQueryContext(ctx, r.session).
		BindStruct(scylla.MessagesByChatStruct{
			ChatId: [16]byte(chatID),
		})

	if err := q.SelectRelease(&rows); err != nil {
		return nil, err
	}

	msgs := make([]dto.Message, len(rows))

	for i, row := range rows {
		msgs[i] = toMessage(row)
	}

	return msgs, nil
}

func (r *messageRepo) Delete(ctx context.Context, chatID uuid.UUID, createdAt time.Time, id int64) error {
	q := scylla.MessagesByChat.
		DeleteQueryContext(ctx, r.session).
		BindStruct(scylla.MessagesByChatStruct{
            ChatId:    [16]byte(chatID),
            CreatedAt: createdAt,
            Id:        id,
        })

	return q.ExecRelease()
}

func toMessage(m scylla.MessagesByChatStruct) dto.Message {
	var quotedID *int64
	if m.QuotedId != 0 {
		quotedID = &m.QuotedId
	}

	return dto.Message{
        ChatID:    uuid.UUID(m.ChatId),
        CreatedAt: m.CreatedAt,
        ID:        m.Id,
        SenderID:  uuid.UUID(m.SenderId),
        Data:      m.Data,
        DataType:  m.DataType,
        QuotedID:  quotedID,
    }
}