package repository

import (
	"context"
	"spotify-chat/internal/db/scylla"
	"spotify-chat/internal/dto"
	"strings"
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
		row.QuotedId = *params.QuotedID
	}

	if err := scylla.MessagesByChat.
		InsertQueryContext(ctx, r.session).
		BindStruct(row).
		ExecRelease(); err != nil {
		return dto.Message{}, err
	}

	return toMessage(row), nil
}

func (r *messageRepo) GetByChatID(ctx context.Context, chatID uuid.UUID, limit int32, timeOffset *time.Time) ([]dto.Message, error) {
	var rows []scylla.MessagesByChatStruct

	b := qb.Select("messages_by_chat").
		Where(qb.Eq("chat_id"))

	row := scylla.MessagesByChatStruct{ChatId: chatID}
	if timeOffset != nil {
		b = b.Where(qb.Lt("created_at"))
		row.CreatedAt = *timeOffset
	}

	q := b.
		OrderBy("created_at", qb.DESC).
		LimitNamed("limit").
		QueryContext(ctx, r.session).
		BindStructMap(row, map[string]any{
			"limit": limit,
		})

	if err := q.SelectRelease(&rows); err != nil {
		return nil, err
	}

	return toMessages(rows), nil
}

func (r *messageRepo) GetLastMessagesByChatIDs(ctx context.Context, chatIDs []uuid.UUID) (map[uuid.UUID]dto.Message, error) {
	if len(chatIDs) == 0 {
		return map[uuid.UUID]dto.Message{}, nil
	}

	stmt := "SELECT * FROM messages_by_chat WHERE " +
		inClause("chat_id", len(chatIDs)) +
		" PER PARTITION LIMIT 1"

	args := make([]any, len(chatIDs))
	for i, id := range chatIDs {
		args[i] = id
	}

	var rows []scylla.MessagesByChatStruct

	if err := r.session.Query(stmt, nil).
		WithContext(ctx).
		Bind(args...).
		SelectRelease(&rows); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]dto.Message, len(rows))
	for _, row := range rows {
		msg := toMessage(row)
		result[msg.ChatID] = msg
	}

	return result, nil
}


func inClause(column string, n int) string {
	return column + " IN (" + strings.Repeat("?,", n-1) + "?)"
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
