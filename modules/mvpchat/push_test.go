package mvpchat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebPushNotifierDispatchesPayload(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	n := &WebPushNotifier{dispatchURL: srv.URL, apiKey: "k", httpClient: srv.Client()}
	err := n.NotifyMessage(context.Background(), []PushSubscription{{Endpoint: "e", P256DH: "p", Auth: "a"}}, "t", "b", "chat-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatalf("expected dispatch request")
	}
}

func TestWebPushNotifierDisabled(t *testing.T) {
	n := &WebPushNotifier{}
	if err := n.NotifyMessage(context.Background(), []PushSubscription{{Endpoint: "e"}}, "t", "b", "c"); err != nil {
		t.Fatalf("expected nil err on disabled notifier, got %v", err)
	}
}
