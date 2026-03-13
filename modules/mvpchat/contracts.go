package mvpchat

import "time"

type ChatListItem struct {
	ChatID         string    `json:"chat_id"`
	ContactUserID  string    `json:"contact_user_id"`
	ContactName    string    `json:"contact_name"`
	LastMessage    string    `json:"last_message"`
	LastMessageAt  time.Time `json:"last_message_at"`
	HasLastMessage bool      `json:"has_last_message"`
}

type Message struct {
	ID            string    `json:"id"`
	ChatID        string    `json:"chat_id"`
	SenderUserID  string    `json:"sender_user_id"`
	SenderDisplay string    `json:"sender_display"`
	Content       string    `json:"content"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type QRToken struct {
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ImageDataURL string    `json:"image_data_url"`
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
