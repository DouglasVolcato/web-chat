package mvpchat

import "sync"

type messageHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan struct{}]struct{}
}

func newMessageHub() *messageHub {
	return &messageHub{subscribers: make(map[string]map[chan struct{}]struct{})}
}

func (h *messageHub) Subscribe(userID string) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)

	h.mu.Lock()
	if h.subscribers[userID] == nil {
		h.subscribers[userID] = make(map[chan struct{}]struct{})
	}
	h.subscribers[userID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		subs := h.subscribers[userID]
		if subs == nil {
			return
		}

		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
		}

		if len(subs) == 0 {
			delete(h.subscribers, userID)
		}
	}

	return ch, cancel
}

func (h *messageHub) Publish(userIDs ...string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		for ch := range h.subscribers[userID] {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
}
