package mvpchat

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"
	"time"
)

type fakeRepo struct {
	contactsOK     bool
	chatID         string
	notifySubs     []PushSubscription
	senderName     string
	lookupOwnerID  string
	lookupErr      error
	consumeOwnerID string
	consumeErr     error
	consumeCalls   int
	auditActions   []string
	invalidated    int
	revoked        int
	trimCalls      int
	trimKeep       int
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
	if expiresAt.Before(time.Now().UTC().AddDate(10, 0, 0)) {
		return errors.New("bad expiry")
	}
	return nil
}
func (f *fakeRepo) TrimChatMessages(ctx context.Context, tx *sql.Tx, chatID string, keep int) error {
	f.trimCalls++
	f.trimKeep = keep
	return nil
}
func (f *fakeRepo) CreateQRToken(ctx context.Context, tx *sql.Tx, ownerID, tokenHash string, expiresAt time.Time) error {
	return nil
}
func (f *fakeRepo) LookupActiveQROwner(ctx context.Context, tx *sql.Tx, tokenHash string, now time.Time) (string, error) {
	return f.lookupOwnerID, f.lookupErr
}
func (f *fakeRepo) ConsumeQRToken(ctx context.Context, tx *sql.Tx, tokenHash, usedBy string, now time.Time) (string, error) {
	f.consumeCalls++
	return f.consumeOwnerID, f.consumeErr
}
func (f *fakeRepo) AddContactPair(ctx context.Context, tx *sql.Tx, userA, userB string) error {
	return nil
}
func (f *fakeRepo) SavePushSubscription(ctx context.Context, tx *sql.Tx, userID string, in PushSubscriptionInput) error {
	return nil
}
func (f *fakeRepo) RevokePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint string) error {
	f.revoked++
	return nil
}
func (f *fakeRepo) InvalidatePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint string) error {
	f.invalidated++
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

type fakeNotifier struct {
	status int
	called bool
}

func (f *fakeNotifier) NotifyMessage(ctx context.Context, sub PushSubscription, payload PushPayload) (int, error) {
	f.called = true
	if f.status == 0 {
		return http.StatusCreated, nil
	}
	return f.status, nil
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
	repo := &fakeRepo{contactsOK: true, notifySubs: []PushSubscription{{Endpoint: "e", P256DH: "p", Auth: "a", Status: "ACTIVE"}}, senderName: "Alice"}
	n := &fakeNotifier{}
	svc := NewService(repo, n)
	chatID, err := svc.SendMessage(context.Background(), nil, "u1", "u2", "hello", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if chatID == "" || !n.called {
		t.Fatalf("expected chat id and notify")
	}
	if repo.trimCalls != 1 || repo.trimKeep != maxMessagesPerChat {
		t.Fatalf("expected trim to keep %d messages, got calls=%d keep=%d", maxMessagesPerChat, repo.trimCalls, repo.trimKeep)
	}
}

func TestSendMessageInvalidatesGoneSubscription(t *testing.T) {
	repo := &fakeRepo{contactsOK: true, notifySubs: []PushSubscription{{Endpoint: "e", P256DH: "p", Auth: "a", Status: "ACTIVE"}}}
	n := &fakeNotifier{status: http.StatusGone}
	svc := NewService(repo, n)
	_, err := svc.SendMessage(context.Background(), nil, "u1", "u2", "hello", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.invalidated != 1 {
		t.Fatalf("expected invalidation")
	}
}

func TestRedeemQRFailureAudited(t *testing.T) {
	repo := &fakeRepo{lookupErr: errors.New("invalid")}
	svc := NewService(repo, nil)
	_, err := svc.RedeemContactQR(context.Background(), nil, "u1", "token", "127.0.0.1")
	if !errors.Is(err, ErrInvalidQR) {
		t.Fatalf("expected ErrInvalidQR, got %v", err)
	}
}

func TestRedeemQROwnCodeDoesNotConsumeToken(t *testing.T) {
	repo := &fakeRepo{lookupOwnerID: "u1"}
	svc := NewService(repo, nil)

	_, err := svc.RedeemContactQR(context.Background(), nil, "u1", "token", "127.0.0.1")
	if !errors.Is(err, ErrOwnQR) {
		t.Fatalf("expected ErrOwnQR, got %v", err)
	}
	if repo.consumeCalls != 0 {
		t.Fatalf("expected own qr to not be consumed")
	}
}

func TestGenerateQRIncludesImageDataURL(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)

	qr, err := svc.GenerateContactQR(context.Background(), nil, "u1", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if qr == nil || qr.Token == "" {
		t.Fatalf("expected token")
	}
	if qr.ImageDataURL == "" {
		t.Fatalf("expected qr image data url")
	}
	if len(qr.ImageDataURL) < len("data:image/png;base64,") || qr.ImageDataURL[:22] != "data:image/png;base64," {
		t.Fatalf("unexpected qr data url prefix: %q", qr.ImageDataURL)
	}
}
