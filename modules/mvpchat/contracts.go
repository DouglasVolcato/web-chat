package mvpchat

import "time"

type ChatListItem struct {
	ChatID         string
	ContactUserID  string
	ContactName    string
	LastMessage    string
	LastMessageAt  time.Time
	HasLastMessage bool
}

type Message struct {
	ID            string
	ChatID        string
	SenderUserID  string
	SenderDisplay string
	Content       string
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

type QRToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PushSubscriptionInput struct {
	Endpoint  string
	P256DH    string
	Auth      string
	Device    string
	UserAgent string
}

type PushSubscription struct {
	Endpoint  string
	P256DH    string
	Auth      string
	Status    string
	RevokedAt time.Time
}

type PushPayload struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	Tag        string `json:"tag"`
	ChatID     string `json:"chat_id"`
	URL        string `json:"url"`
	Timestamp  int64  `json:"timestamp"`
	SenderName string `json:"sender_name,omitempty"`
}
