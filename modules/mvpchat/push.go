package mvpchat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

type PushNotifier interface {
	NotifyMessage(ctx context.Context, subs []PushSubscription, title, body, chatID string) error
}

type WebPushNotifier struct {
	dispatchURL string
	apiKey      string
	httpClient  *http.Client
}

type dispatchPayload struct {
	Title         string             `json:"title"`
	Body          string             `json:"body"`
	ChatID        string             `json:"chat_id"`
	URL           string             `json:"url"`
	Subscriptions []PushSubscription `json:"subscriptions"`
}

func NewWebPushNotifierFromEnv() *WebPushNotifier {
	return &WebPushNotifier{
		dispatchURL: strings.TrimSpace(os.Getenv("PUSH_DISPATCH_URL")),
		apiKey:      strings.TrimSpace(os.Getenv("PUSH_DISPATCH_API_KEY")),
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (n *WebPushNotifier) enabled() bool {
	return n != nil && n.dispatchURL != ""
}

func (n *WebPushNotifier) NotifyMessage(ctx context.Context, subs []PushSubscription, title, body, chatID string) error {
	if !n.enabled() || len(subs) == 0 {
		return nil
	}

	payload, err := json.Marshal(dispatchPayload{
		Title:         strings.TrimSpace(title),
		Body:          strings.TrimSpace(body),
		ChatID:        strings.TrimSpace(chatID),
		URL:           "/app/messages/" + strings.TrimSpace(chatID),
		Subscriptions: subs,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.dispatchURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if n.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+n.apiKey)
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &PushDispatchError{StatusCode: resp.StatusCode}
	}

	return nil
}

type PushDispatchError struct {
	StatusCode int
}

func (e *PushDispatchError) Error() string {
	return http.StatusText(e.StatusCode)
}
