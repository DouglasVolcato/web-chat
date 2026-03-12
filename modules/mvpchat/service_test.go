package mvpchat

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type fakeRepo struct {
	contactsOK       bool
	chatID           string
	notifySubs       []PushSubscription
	senderName       string
	consumeOwnerID   string
	consumeErr       error
	addContactCalled bool
	auditActions     []string
}

func (f *fakeRepo) PurgeExpiredMessages(ctx context.Context, tx *sql.Tx) error { return nil }
func (f *fakeRepo) GetUserChats(ctx context.Context, tx *sql.Tx, userID string) ([]ChatListItem, error) {
	return nil, nil
}
func (f *fakeRepo) GetChatMessages(ctx context.Context, tx *sql.Tx, userID, chatID string) ([]Message, error) {
	return nil, nil
}
func (f *fakeRepo) AreContacts(ctx context.Context, tx *sql.Tx, userID, targetID string) (bool, error) {
	return f.contactsOK, nil
}
func (f *fakeRepo) GetOrCreateDirectChat(ctx context.Context, tx *sql.Tx, userA, userB string) (string, error) {
	if f.chatID == "" {
		f.chatID = "chat-1"
	}
	return f.chatID, nil
}
func (f *fakeRepo) CreateMessage(ctx context.Context, tx *sql.Tx, chatID, senderID, content string, expiresAt time.Time) error {
	if expiresAt.Before(time.Now().UTC().Add(23 * time.Hour)) {
		return errors.New("bad expiry")
	}
	return nil
}
func (f *fakeRepo) CreateQRToken(ctx context.Context, tx *sql.Tx, ownerID, tokenHash string, expiresAt time.Time) error {
	return nil
}
func (f *fakeRepo) ConsumeQRToken(ctx context.Context, tx *sql.Tx, tokenHash, usedBy string, now time.Time) (string, error) {
	return f.consumeOwnerID, f.consumeErr
}
func (f *fakeRepo) AddContactPair(ctx context.Context, tx *sql.Tx, userA, userB string) error {
	f.addContactCalled = true
	return nil
}
func (f *fakeRepo) SavePushSubscription(ctx context.Context, tx *sql.Tx, userID string, in PushSubscriptionInput) error {
	return nil
}
func (f *fakeRepo) ListActivePushSubscriptions(ctx context.Context, tx *sql.Tx, userID string) ([]PushSubscription, error) {
	return f.notifySubs, nil
}
func (f *fakeRepo) GetUserDisplayName(ctx context.Context, tx *sql.Tx, userID string) (string, error) {
	return f.senderName, nil
}
func (f *fakeRepo) DeleteUserAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	return nil
}
func (f *fakeRepo) InsertAuditLog(ctx context.Context, tx *sql.Tx, userID, action, ip string, metadata map[string]any) error {
	f.auditActions = append(f.auditActions, action)
	return nil
}

type fakeNotifier struct{ called bool }

func (f *fakeNotifier) NotifyMessage(ctx context.Context, subs []PushSubscription, title, body, chatID string) error {
	f.called = true
	return nil
}

func TestSendMessageRequiresContact(t *testing.T) {
	repo := &fakeRepo{contactsOK: false}
	svc := NewService(repo, &fakeNotifier{})
	_, err := svc.SendMessage(context.Background(), nil, "u1", "u2", "oi", "127.0.0.1")
	if !errors.Is(err, ErrNotContact) {
		t.Fatalf("expected ErrNotContact, got %v", err)
	}
}

func TestSendMessageNotifiesAndAudits(t *testing.T) {
	repo := &fakeRepo{contactsOK: true, notifySubs: []PushSubscription{{Endpoint: "e", P256DH: "p", Auth: "a"}}, senderName: "Alice"}
	n := &fakeNotifier{}
	svc := NewService(repo, n)
	chatID, err := svc.SendMessage(context.Background(), nil, "u1", "u2", "hello", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if chatID == "" {
		t.Fatalf("expected chat id")
	}
	if !n.called {
		t.Fatalf("expected notifier call")
	}
	if len(repo.auditActions) == 0 || repo.auditActions[0] != "message.sent" {
		t.Fatalf("expected message.sent audit")
	}
}

func TestRedeemQRFailureAudited(t *testing.T) {
	repo := &fakeRepo{consumeErr: errors.New("invalid")}
	svc := NewService(repo, nil)
	_, err := svc.RedeemContactQR(context.Background(), nil, "u1", "token", "127.0.0.1")
	if !errors.Is(err, ErrInvalidQR) {
		t.Fatalf("expected ErrInvalidQR, got %v", err)
	}
	found := false
	for _, a := range repo.auditActions {
		if a == "contact.qr.failed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected contact.qr.failed audit")
	}
}
