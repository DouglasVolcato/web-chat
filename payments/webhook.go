package payments

import (
	"context"
	"encoding/json"
	"fmt"
)

// HandleWebhookPayload parses and dispatches webhook events.
func (s *Service) HandleWebhookPayload(ctx context.Context, payload []byte) error {
	var event NotificationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("payload inv\u00e1lido")
	}
	return s.HandleWebhookNotification(ctx, event)
}
