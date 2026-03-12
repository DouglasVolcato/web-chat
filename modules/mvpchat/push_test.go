package mvpchat

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebPushNotifierDirectSend(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST")
		}
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("expected VAPID auth header")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	priv := testVAPIDPrivateKey(t)
	n := &WebPushNotifier{publicKey: "pub", privateKey: priv, subject: "mailto:test@example.com", client: srv.Client()}
	status, err := n.NotifyMessage(context.Background(), PushSubscription{Endpoint: srv.URL, P256DH: "p", Auth: "a", Status: "ACTIVE"}, PushPayload{Title: "t", Body: "b", URL: "/app/messages/c1"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status != http.StatusCreated || !called {
		t.Fatalf("expected success")
	}
}

func TestWebPushNotifierDisabled(t *testing.T) {
	n := &WebPushNotifier{}
	status, err := n.NotifyMessage(context.Background(), PushSubscription{Endpoint: "https://example.com"}, PushPayload{Title: "x"})
	if err != nil || status != 0 {
		t.Fatalf("expected disabled no-op")
	}
}

func testVAPIDPrivateKey(t *testing.T) string {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	d := k.D.FillBytes(make([]byte, 32))
	return base64.RawURLEncoding.EncodeToString(d)
}
