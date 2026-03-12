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
	Token     string
	ExpiresAt time.Time
}

type PushSubscriptionInput struct {
	Endpoint string
	P256DH   string
	Auth     string
}

type PushSubscription struct {
	Endpoint string
	P256DH   string
	Auth     string
}
