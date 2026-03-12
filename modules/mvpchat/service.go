package mvpchat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

var (
	ErrInvalidMessage = errors.New("mensagem inválida")
	ErrNotContact     = errors.New("usuário não é seu contato")
	ErrInvalidQR      = errors.New("qr inválido ou expirado")
)

type Service struct {
	repo     Repository
	notifier PushNotifier
}

func NewService(repo Repository, notifier PushNotifier) *Service {
	return &Service{repo: repo, notifier: notifier}
}

func (s *Service) PurgeExpiredMessages(ctx context.Context, tx *sql.Tx) error {
	return s.repo.PurgeExpiredMessages(ctx, tx)
}

func (s *Service) ListChats(ctx context.Context, tx *sql.Tx, userID string) ([]ChatListItem, error) {
	if err := s.repo.PurgeExpiredMessages(ctx, tx); err != nil {
		return nil, err
	}
	return s.repo.GetUserChats(ctx, tx, userID)
}

func (s *Service) ListMessages(ctx context.Context, tx *sql.Tx, userID, chatID string) ([]Message, error) {
	if err := s.repo.PurgeExpiredMessages(ctx, tx); err != nil {
		return nil, err
	}
	return s.repo.GetChatMessages(ctx, tx, userID, chatID)
}

func (s *Service) SendMessage(ctx context.Context, tx *sql.Tx, senderID, targetID, content, ip string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 2000 {
		return "", ErrInvalidMessage
	}
	ok, err := s.repo.AreContacts(ctx, tx, senderID, targetID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotContact
	}
	chatID, err := s.repo.GetOrCreateDirectChat(ctx, tx, senderID, targetID)
	if err != nil {
		return "", err
	}
	if err := s.repo.CreateMessage(ctx, tx, chatID, senderID, content, time.Now().UTC().Add(24*time.Hour)); err != nil {
		return "", err
	}

	if s.notifier != nil {
		subs, _ := s.repo.ListActivePushSubscriptions(ctx, tx, targetID)
		senderName, _ := s.repo.GetUserDisplayName(ctx, tx, senderID)
		_ = s.notifier.NotifyMessage(ctx, subs, defaultIfEmpty(senderName, "Nova mensagem"), content, chatID)
	}

	_ = s.repo.InsertAuditLog(ctx, tx, senderID, "message.sent", sanitizeIP(ip), map[string]any{"chat_id": chatID, "target_user_id": targetID})
	return chatID, nil
}

func (s *Service) GenerateContactQR(ctx context.Context, tx *sql.Tx, ownerID, ip string) (*QRToken, error) {
	token, hash, err := generateSecureToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().UTC().Add(5 * time.Minute)
	if err := s.repo.CreateQRToken(ctx, tx, ownerID, hash, expires); err != nil {
		return nil, err
	}
	_ = s.repo.InsertAuditLog(ctx, tx, ownerID, "contact.qr.generated", sanitizeIP(ip), nil)
	return &QRToken{Token: token, ExpiresAt: expires}, nil
}

func (s *Service) RedeemContactQR(ctx context.Context, tx *sql.Tx, userID, token, ip string) (string, error) {
	hash := hashToken(token)
	ownerID, err := s.repo.ConsumeQRToken(ctx, tx, hash, userID, time.Now().UTC())
	if err != nil || ownerID == "" || ownerID == userID {
		_ = s.repo.InsertAuditLog(ctx, tx, userID, "contact.qr.failed", sanitizeIP(ip), nil)
		return "", ErrInvalidQR
	}
	if err := s.repo.AddContactPair(ctx, tx, ownerID, userID); err != nil {
		return "", err
	}
	chatID, err := s.repo.GetOrCreateDirectChat(ctx, tx, ownerID, userID)
	if err != nil {
		return "", err
	}
	_ = s.repo.InsertAuditLog(ctx, tx, userID, "contact.qr.redeemed", sanitizeIP(ip), map[string]any{"owner_user_id": ownerID, "chat_id": chatID})
	return chatID, nil
}

func (s *Service) SavePushSubscription(ctx context.Context, tx *sql.Tx, userID, ip string, in PushSubscriptionInput) error {
	if strings.TrimSpace(in.Endpoint) == "" || strings.TrimSpace(in.P256DH) == "" || strings.TrimSpace(in.Auth) == "" {
		return errors.New("subscription inválida")
	}
	if err := s.repo.SavePushSubscription(ctx, tx, userID, in); err != nil {
		return err
	}
	return s.repo.InsertAuditLog(ctx, tx, userID, "push.subscription.saved", sanitizeIP(ip), nil)
}

func (s *Service) DeleteAccount(ctx context.Context, tx *sql.Tx, userID, ip string) error {
	if err := s.repo.DeleteUserAccount(ctx, tx, userID); err != nil {
		return err
	}
	return s.repo.InsertAuditLog(ctx, tx, userID, "account.deleted", sanitizeIP(ip), nil)
}

func hashToken(token string) string {
	s := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return fmt.Sprintf("%x", s[:])
}

func generateSecureToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, hashToken(token), nil
}

func sanitizeIP(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

func defaultIfEmpty(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}
