package mvpchat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

var (
	ErrInvalidMessage = errors.New("mensagem inválida")
	ErrNotContact     = errors.New("usuário não é seu contato")
	ErrChatForbidden  = errors.New("chat inválido")
	ErrInvalidQR      = errors.New("qr inválido ou expirado")
	ErrOwnQR          = errors.New("você não pode usar seu próprio QR Code")
)

const qrCodeImageSize = 280
const maxMessagesPerChat = 15

type Service struct {
	repo     Repository
	notifier PushNotifier
	logger   *slog.Logger
}

func NewService(repo Repository, notifier PushNotifier) *Service {
	return &Service{repo: repo, notifier: notifier, logger: slog.Default()}
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

func (s *Service) ClearChat(ctx context.Context, tx *sql.Tx, userID, chatID, ip string) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return ErrChatForbidden
	}

	ok, err := s.repo.IsChatParticipant(ctx, tx, userID, chatID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrChatForbidden
	}

	if err := s.repo.DeleteChatMessages(ctx, tx, chatID); err != nil {
		return err
	}

	return s.repo.InsertAuditLog(ctx, tx, userID, "chat.cleared", sanitizeIP(ip), map[string]any{"chat_id": chatID})
}

func (s *Service) RegisterPushSubscription(ctx context.Context, tx *sql.Tx, userID, ip string, in PushSubscriptionInput) error {
	if strings.TrimSpace(in.Endpoint) == "" || strings.TrimSpace(in.P256DH) == "" || strings.TrimSpace(in.Auth) == "" {
		return errors.New("subscription inválida")
	}
	if err := s.repo.SavePushSubscription(ctx, tx, userID, in); err != nil {
		return err
	}
	s.logger.Info("push_subscription_registered", "user_id", userID)
	return s.repo.InsertAuditLog(ctx, tx, userID, "push.subscription.saved", sanitizeIP(ip), nil)
}

func (s *Service) RevokePushSubscription(ctx context.Context, tx *sql.Tx, userID, endpoint, ip string) error {
	if strings.TrimSpace(endpoint) == "" {
		return errors.New("endpoint obrigatório")
	}
	if err := s.repo.RevokePushSubscription(ctx, tx, userID, endpoint); err != nil {
		return err
	}
	s.logger.Info("push_subscription_revoked", "user_id", userID)
	return s.repo.InsertAuditLog(ctx, tx, userID, "push.subscription.revoked", sanitizeIP(ip), nil)
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
	if err := s.repo.CreateMessage(ctx, tx, chatID, senderID, content, time.Now().UTC().AddDate(20, 0, 0)); err != nil {
		return "", err
	}
	if err := s.repo.TrimChatMessages(ctx, tx, chatID, maxMessagesPerChat); err != nil {
		return "", err
	}

	s.trySendPush(ctx, tx, senderID, targetID, chatID, content)
	_ = s.repo.InsertAuditLog(ctx, tx, senderID, "message.sent", sanitizeIP(ip), map[string]any{"chat_id": chatID, "target_user_id": targetID})
	return chatID, nil
}

func (s *Service) trySendPush(ctx context.Context, tx *sql.Tx, senderID, targetID, chatID, content string) {
	if s.notifier == nil {
		return
	}
	subs, err := s.repo.ListActivePushSubscriptions(ctx, tx, targetID)
	if err != nil {
		s.logger.Error("push_send_failed", "reason", err.Error())
		return
	}
	if len(subs) == 0 {
		return
	}
	senderName, _ := s.repo.GetUserDisplayName(ctx, tx, senderID)
	payload := PushPayload{
		Title:      defaultIfEmpty(senderName, "Nova mensagem"),
		Body:       summarizePushMessage(content),
		Tag:        "messages-chat-" + chatID,
		ChatID:     chatID,
		URL:        "/app/messages/" + chatID,
		Timestamp:  time.Now().UnixMilli(),
		SenderName: senderName,
	}
	s.logger.Info("push_send_started", "chat_id", chatID, "subscriptions", len(subs))

	for _, sub := range subs {
		status, err := s.notifier.NotifyMessage(ctx, sub, payload)
		if err != nil {
			s.logger.Error("push_send_failed", "chat_id", chatID, "error", err.Error())
			continue
		}
		if status == http.StatusGone || status == http.StatusNotFound {
			_ = s.repo.InvalidatePushSubscription(ctx, tx, targetID, sub.Endpoint)
			s.logger.Warn("push_subscription_invalidated", "chat_id", chatID)
			continue
		}
		if status >= 200 && status < 300 {
			s.logger.Info("push_send_succeeded", "chat_id", chatID)
			continue
		}
		s.logger.Error("push_send_failed", "chat_id", chatID, "status", status)
	}
}

func summarizePushMessage(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "Voce recebeu uma nova mensagem."
	}
	const maxLen = 120
	runes := []rune(trimmed)
	if len(runes) <= maxLen {
		return trimmed
	}
	return string(runes[:maxLen-1]) + "…"
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
	imageDataURL, err := buildQRCodeDataURL(token)
	if err != nil {
		return nil, err
	}
	_ = s.repo.InsertAuditLog(ctx, tx, ownerID, "contact.qr.generated", sanitizeIP(ip), nil)
	return &QRToken{Token: token, ExpiresAt: expires, ImageDataURL: imageDataURL}, nil
}

func (s *Service) RedeemContactQR(ctx context.Context, tx *sql.Tx, userID, token, ip string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		_ = s.repo.InsertAuditLog(ctx, tx, userID, "contact.qr.failed", sanitizeIP(ip), map[string]any{"reason": "empty_token"})
		return "", ErrInvalidQR
	}

	now := time.Now().UTC()
	hash := hashToken(token)
	ownerID, err := s.repo.LookupActiveQROwner(ctx, tx, hash, now)
	if err != nil || ownerID == "" {
		_ = s.repo.InsertAuditLog(ctx, tx, userID, "contact.qr.failed", sanitizeIP(ip), nil)
		return "", ErrInvalidQR
	}
	if ownerID == userID {
		_ = s.repo.InsertAuditLog(ctx, tx, userID, "contact.qr.failed", sanitizeIP(ip), map[string]any{"reason": "own_qr"})
		return "", ErrOwnQR
	}
	consumedOwnerID, err := s.repo.ConsumeQRToken(ctx, tx, hash, userID, now)
	if err != nil || consumedOwnerID == "" {
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

func (s *Service) DeleteAccount(ctx context.Context, tx *sql.Tx, userID, ip string) error {
	if err := s.repo.DeleteUserAccount(ctx, tx, userID); err != nil {
		return err
	}
	return s.repo.InsertAuditLog(ctx, tx, userID, "account.deleted", sanitizeIP(ip), nil)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return fmt.Sprintf("%x", sum[:])
}

func generateSecureToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, hashToken(token), nil
}

func buildQRCodeDataURL(data string) (string, error) {
	png, err := qrcode.Encode(data, qrcode.Medium, qrCodeImageSize)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
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
