package helpers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUserRateLimitScopesAreIndependent(t *testing.T) {
	rateMu.Lock()
	userWindows = map[string]userWindow{}
	rateMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(
		context.WithValue(context.Background(), userContextKey, "user-1"),
	)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	messagesLimiter := UserRateLimit("messages", 1, time.Minute)(okHandler)
	pushLimiter := UserRateLimit("push", 1, time.Minute)(okHandler)

	firstMessages := httptest.NewRecorder()
	messagesLimiter.ServeHTTP(firstMessages, req)
	if firstMessages.Code != http.StatusNoContent {
		t.Fatalf("first messages request = %d; want %d", firstMessages.Code, http.StatusNoContent)
	}

	firstPush := httptest.NewRecorder()
	pushLimiter.ServeHTTP(firstPush, req)
	if firstPush.Code != http.StatusNoContent {
		t.Fatalf("first push request = %d; want %d", firstPush.Code, http.StatusNoContent)
	}

	secondMessages := httptest.NewRecorder()
	messagesLimiter.ServeHTTP(secondMessages, req)
	if secondMessages.Code != http.StatusTooManyRequests {
		t.Fatalf("second messages request = %d; want %d", secondMessages.Code, http.StatusTooManyRequests)
	}
}
