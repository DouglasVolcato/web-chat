package helpers

import (
	"net/http"
	"sync"
	"time"
)

type userWindow struct {
	Count   int
	ResetAt time.Time
}

var (
	rateMu      sync.Mutex
	userWindows = map[string]userWindow{}
)

func UserRateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := ResolveUserIDFromRequest(r)
			if err != nil || userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			now := time.Now().UTC()
			rateMu.Lock()
			state := userWindows[userID]
			if state.ResetAt.IsZero() || now.After(state.ResetAt) {
				state = userWindow{Count: 0, ResetAt: now.Add(window)}
			}
			state.Count++
			userWindows[userID] = state
			rateMu.Unlock()

			if state.Count > limit {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limit por usuário excedido"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
