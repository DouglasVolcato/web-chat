package mvpchat

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"app/models"
)

type Repository interface {
	PurgeExpiredMessages(ctx context.Context, tx *sql.Tx) error
	GetUserChats(ctx context.Context, tx *sql.Tx, userID string) ([]ChatListItem, error)
	GetChatMessages(ctx context.Context, tx *sql.Tx, userID, chatID string) ([]Message, error)
	AreContacts(ctx context.Context, tx *sql.Tx, userID, targetID string) (bool, error)
	GetOrCreateDirectChat(ctx context.Context, tx *sql.Tx, userA, userB string) (string, error)
	CreateMessage(ctx context.Context, tx *sql.Tx, chatID, senderID, content string, expiresAt time.Time) error
	CreateQRToken(ctx context.Context, tx *sql.Tx, ownerID, tokenHash string, expiresAt time.Time) error
	ConsumeQRToken(ctx context.Context, tx *sql.Tx, tokenHash, usedBy string, now time.Time) (string, error)
	AddContactPair(ctx context.Context, tx *sql.Tx, userA, userB string) error
	SavePushSubscription(ctx context.Context, tx *sql.Tx, userID string, in PushSubscriptionInput) error
	RevokePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint string) error
	InvalidatePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint string) error
	ListActivePushSubscriptions(ctx context.Context, tx *sql.Tx, userID string) ([]PushSubscription, error)
	GetUserDisplayName(ctx context.Context, tx *sql.Tx, userID string) (string, error)
	DeleteUserAccount(ctx context.Context, tx *sql.Tx, userID string) error
	InsertAuditLog(ctx context.Context, tx *sql.Tx, userID, action, ip string, metadata map[string]any) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) PurgeExpiredMessages(ctx context.Context, tx *sql.Tx) error {
	_, err := models.ExecContext(tx, ctx, `delete from messages where expires_at <= now()`)
	return err
}

func (r *PostgresRepository) GetUserChats(ctx context.Context, tx *sql.Tx, userID string) ([]ChatListItem, error) {
	rows, err := models.QueryContext(tx, ctx, `
		select c.id, u.id, u.name,
		coalesce(m.content, '') as last_message,
		m.created_at
		from chats c
		join chat_participants p1 on p1.chat_id = c.id and p1.user_id = $1
		join chat_participants p2 on p2.chat_id = c.id and p2.user_id <> $1
		join users u on u.id = p2.user_id and u.deleted_at is null
		left join lateral (
			select content, created_at from messages
			where chat_id = c.id and expires_at > now()
			order by created_at desc
			limit 1
		) m on true
		order by m.created_at desc nulls last, c.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ChatListItem, 0)
	for rows.Next() {
		var it ChatListItem
		var lastAt sql.NullTime
		if err := rows.Scan(&it.ChatID, &it.ContactUserID, &it.ContactName, &it.LastMessage, &lastAt); err != nil {
			return nil, err
		}
		if lastAt.Valid {
			it.LastMessageAt = lastAt.Time
			it.HasLastMessage = true
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) GetChatMessages(ctx context.Context, tx *sql.Tx, userID, chatID string) ([]Message, error) {
	rows, err := models.QueryContext(tx, ctx, `
		select m.id,m.chat_id,m.sender_user_id,u.name,m.content,m.created_at,m.expires_at
		from messages m
		join users u on u.id = m.sender_user_id
		where m.chat_id = $1 and m.expires_at > now() and exists (
			select 1 from chat_participants p where p.chat_id = m.chat_id and p.user_id = $2
		)
		order by m.created_at asc
	`, chatID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderUserID, &m.SenderDisplay, &m.Content, &m.CreatedAt, &m.ExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) AreContacts(ctx context.Context, tx *sql.Tx, userID, targetID string) (bool, error) {
	row := models.QueryRowContext(tx, ctx, `select exists(select 1 from contacts where user_id=$1 and contact_user_id=$2)`, userID, targetID)
	var ok bool
	if err := row.Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func (r *PostgresRepository) GetOrCreateDirectChat(ctx context.Context, tx *sql.Tx, userA, userB string) (string, error) {
	row := models.QueryRowContext(tx, ctx, `select p1.chat_id from chat_participants p1 join chat_participants p2 on p2.chat_id = p1.chat_id where p1.user_id=$1 and p2.user_id=$2 limit 1`, userA, userB)
	var chatID string
	if err := row.Scan(&chatID); err == nil {
		return chatID, nil
	}
	if err := models.QueryRowContext(tx, ctx, `insert into chats default values returning id`).Scan(&chatID); err != nil {
		return "", err
	}
	if _, err := models.ExecContext(tx, ctx, `insert into chat_participants(chat_id,user_id) values ($1,$2),($1,$3)`, chatID, userA, userB); err != nil {
		return "", err
	}
	return chatID, nil
}

func (r *PostgresRepository) CreateMessage(ctx context.Context, tx *sql.Tx, chatID, senderID, content string, expiresAt time.Time) error {
	_, err := models.ExecContext(tx, ctx, `insert into messages(chat_id,sender_user_id,content,expires_at) values ($1,$2,$3,$4)`, chatID, senderID, content, expiresAt)
	return err
}

func (r *PostgresRepository) CreateQRToken(ctx context.Context, tx *sql.Tx, ownerID, tokenHash string, expiresAt time.Time) error {
	_, err := models.ExecContext(tx, ctx, `insert into qr_tokens(owner_user_id,token_hash,expires_at) values ($1,$2,$3)`, ownerID, tokenHash, expiresAt)
	return err
}

func (r *PostgresRepository) ConsumeQRToken(ctx context.Context, tx *sql.Tx, tokenHash, usedBy string, now time.Time) (string, error) {
	row := models.QueryRowContext(tx, ctx, `update qr_tokens set used_by_user_id=$2, used_at=$3 where token_hash=$1 and used_at is null and expires_at > $3 returning owner_user_id`, tokenHash, usedBy, now)
	var owner string
	if err := row.Scan(&owner); err != nil {
		return "", err
	}
	return owner, nil
}

func (r *PostgresRepository) AddContactPair(ctx context.Context, tx *sql.Tx, userA, userB string) error {
	_, err := models.ExecContext(tx, ctx, `insert into contacts(user_id,contact_user_id) values ($1,$2),($2,$1) on conflict do nothing`, userA, userB)
	return err
}

func (r *PostgresRepository) SavePushSubscription(ctx context.Context, tx *sql.Tx, userID string, in PushSubscriptionInput) error {
	_, err := models.ExecContext(tx, ctx, `
		insert into push_subscriptions(user_id, endpoint, p256dh, auth, status, user_agent_hash, device_label, last_seen_at, revoked_at)
		values ($1,$2,$3,$4,'ACTIVE',nullif($5,''),nullif($6,''),now(),null)
		on conflict(user_id, endpoint)
		do update set p256dh=excluded.p256dh, auth=excluded.auth, status='ACTIVE', user_agent_hash=excluded.user_agent_hash, device_label=excluded.device_label, last_seen_at=now(), revoked_at=null
	`, userID, in.Endpoint, in.P256DH, in.Auth, in.UserAgent, in.Device)
	return err
}

func (r *PostgresRepository) RevokePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint string) error {
	_, err := models.ExecContext(tx, ctx, `update push_subscriptions set status='REVOKED', revoked_at=now() where user_id=$1 and endpoint=$2`, userID, endpoint)
	return err
}

func (r *PostgresRepository) InvalidatePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint string) error {
	_, err := models.ExecContext(tx, ctx, `update push_subscriptions set status='INVALID', revoked_at=now() where user_id=$1 and endpoint=$2 and status='ACTIVE'`, userID, endpoint)
	return err
}

func (r *PostgresRepository) ListActivePushSubscriptions(ctx context.Context, tx *sql.Tx, userID string) ([]PushSubscription, error) {
	rows, err := models.QueryContext(tx, ctx, `select endpoint,p256dh,auth,status,coalesce(revoked_at,to_timestamp(0)) from push_subscriptions where user_id=$1 and status='ACTIVE' and revoked_at is null`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]PushSubscription, 0)
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.Endpoint, &s.P256DH, &s.Auth, &s.Status, &s.RevokedAt); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) GetUserDisplayName(ctx context.Context, tx *sql.Tx, userID string) (string, error) {
	row := models.QueryRowContext(tx, ctx, `select name from users where id=$1 and deleted_at is null`, userID)
	var name string
	if err := row.Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}

func (r *PostgresRepository) DeleteUserAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	if _, err := models.ExecContext(tx, ctx, `update sessions set revoked_at=now() where user_id=$1 and revoked_at is null`, userID); err != nil {
		return err
	}
	if _, err := models.ExecContext(tx, ctx, `update push_subscriptions set status='REVOKED', revoked_at=now() where user_id=$1 and status='ACTIVE'`, userID); err != nil {
		return err
	}
	if _, err := models.ExecContext(tx, ctx, `update users set deleted_at=now(), email=concat('deleted+',id,'@anon.local'), name='Conta excluída' where id=$1 and deleted_at is null`, userID); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepository) InsertAuditLog(ctx context.Context, tx *sql.Tx, userID, action, ip string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	payload, _ := json.Marshal(metadata)
	_, err := models.ExecContext(tx, ctx, `insert into audit_logs(user_id,action,metadata,ip) values ($1,$2,$3::jsonb,nullif($4,'')::inet)`, strings.TrimSpace(userID), action, string(payload), strings.TrimSpace(ip))
	return err
}
