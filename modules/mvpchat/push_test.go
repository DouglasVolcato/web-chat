package mvpchat

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"io"
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
		if r.Header.Get("Content-Encoding") != "aes128gcm" {
			t.Fatalf("expected aes128gcm content encoding")
		}
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Fatalf("expected binary content type")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("unexpected body read error: %v", err)
		}
		if len(body) < 64 {
			t.Fatalf("expected encrypted payload body")
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	pub, priv := testVAPIDKeyPair(t)
	n := &WebPushNotifier{publicKey: pub, privateKey: priv, subject: "mailto:test@example.com", client: srv.Client()}
	status, err := n.NotifyMessage(context.Background(), PushSubscription{Endpoint: srv.URL, P256DH: testSubscriptionP256DH(t), Auth: testSubscriptionAuth(), Status: "ACTIVE"}, PushPayload{Title: "t", Body: "b", URL: "/app/messages/c1"})
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

func testVAPIDKeyPair(t *testing.T) (string, string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	d := k.D.FillBytes(make([]byte, 32))
	pub := elliptic.Marshal(elliptic.P256(), k.PublicKey.X, k.PublicKey.Y)
	return base64.RawURLEncoding.EncodeToString(pub), base64.RawURLEncoding.EncodeToString(d)
}

func testSubscriptionP256DH(t *testing.T) string {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := elliptic.Marshal(elliptic.P256(), k.PublicKey.X, k.PublicKey.Y)
	return base64.RawURLEncoding.EncodeToString(pub)
}

func testSubscriptionAuth() string {
	secret := make([]byte, 16)
	_, _ = rand.Read(secret)
	return base64.RawURLEncoding.EncodeToString(secret)
}
