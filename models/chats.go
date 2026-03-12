package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type UserChat struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Title     string       `json:"title"`
	Context   string       `json:"context"`
	DeletedAt sql.NullTime `json:"deleted_at"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (c *UserChat) Create(ctx context.Context, tx *sql.Tx) error {
	c.ID = uuid.NewString()

	query := `
        insert into user_chats (
            id,
            user_id,
            title,
            context
        ) values (
            $1,
            $2,
            $3,
            $4
        )
    `

	_, err := ExecContext(
		tx,
		ctx,
		query,
		c.ID,
		c.UserID,
		c.Title,
		c.Context,
	)

	return err
}

func (c *UserChat) Update(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            user_chats
        set
            title = $1,
            context = $2
        where
            id = $3
            and deleted_at is null
    `

	_, err := ExecContext(
		tx,
		ctx,
		query,
		c.Title,
		c.Context,
		c.ID,
	)

	return err
}

func (c *UserChat) Delete(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            user_chats
        set
            title = '',
            context = '',
            deleted_at = NOW()
        where
            id = $1
            and deleted_at is null
    `

	_, err := ExecContext(tx, ctx, query, c.ID)
	return err
}

func GetUserChat(ctx context.Context, tx *sql.Tx, id string) (*UserChat, error) {
	query := `
        select
            id,
            user_id,
            title,
            context,
            deleted_at,
            created_at,
            updated_at
        from user_chats
        where id = $1
            and deleted_at is null
    `

	row := QueryRowContext(tx, ctx, query, id)

	var chat UserChat
	err := row.Scan(
		&chat.ID,
		&chat.UserID,
		&chat.Title,
		&chat.Context,
		&chat.DeletedAt,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

func GetUserChats(ctx context.Context, tx *sql.Tx, userID string) ([]UserChat, error) {
	query := `
        select
            id,
            user_id,
            title,
            context,
            deleted_at,
            created_at,
            updated_at
        from user_chats
        where user_id = $1
            and deleted_at is null
        order by created_at desc
    `

	rows, err := QueryContext(tx, ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []UserChat

	for rows.Next() {
		var chat UserChat

		err = rows.Scan(
			&chat.ID,
			&chat.UserID,
			&chat.Title,
			&chat.Context,
			&chat.DeletedAt,
			&chat.CreatedAt,
			&chat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		chats = append(chats, chat)
	}

	return chats, nil
}

type UserChatMessage struct {
	ID        string       `json:"id"`
	ChatID    string       `json:"chat_id"`
	Role      string       `json:"role"`
	Message   string       `json:"message"`
	Emotion   string       `json:"emotion"`
	DeletedAt sql.NullTime `json:"deleted_at"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (m *UserChatMessage) Create(ctx context.Context, tx *sql.Tx) error {
	m.ID = uuid.NewString()

	query := `
        insert into user_chat_messages (
            id,
            chat_id,
            role,
            message,
            emotion
        ) values (
            $1,
            $2,
            $3,
            $4,
            $5
        )
    `

	_, err := ExecContext(tx, ctx, query, m.ID, m.ChatID, m.Role, m.Message, m.Emotion)
	return err
}

func (m *UserChatMessage) Update(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            user_chat_messages
        set
            role = $1,
            message = $2,
            emotion = $3
        where
            id = $4
            and deleted_at is null
    `

	_, err := ExecContext(tx, ctx, query, m.Role, m.Message, m.Emotion, m.ID)
	return err
}

func (m *UserChatMessage) Delete(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            user_chat_messages
        set
            role = '',
            message = '',
            emotion = '',
            deleted_at = NOW()
        where
            id = $1
            and deleted_at is null
    `

	_, err := ExecContext(tx, ctx, query, m.ID)
	return err
}

func GetUserChatMessage(ctx context.Context, tx *sql.Tx, id string) (*UserChatMessage, error) {
	query := `
        select
            id,
            chat_id,
            role,
            message,
            emotion,
            deleted_at,
            created_at,
            updated_at
        from user_chat_messages
        where id = $1
            and deleted_at is null
    `

	row := QueryRowContext(tx, ctx, query, id)

	var message UserChatMessage
	err := row.Scan(
		&message.ID,
		&message.ChatID,
		&message.Role,
		&message.Message,
		&message.Emotion,
		&message.DeletedAt,
		&message.CreatedAt,
		&message.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &message, nil
}

func GetUserChatMessages(ctx context.Context, tx *sql.Tx, chatID string) ([]UserChatMessage, error) {
	query := `
select
id,
chat_id,
role,
message,
emotion,
deleted_at,
created_at,
updated_at
from user_chat_messages
where chat_id = $1
and deleted_at is null
order by created_at asc
`

	rows, err := QueryContext(tx, ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []UserChatMessage

	for rows.Next() {
		var message UserChatMessage

		err = rows.Scan(
			&message.ID,
			&message.ChatID,
			&message.Role,
			&message.Message,
			&message.Emotion,
			&message.DeletedAt,
			&message.CreatedAt,
			&message.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return messages, nil
}
