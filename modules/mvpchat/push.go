package mvpchat

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PushNotifier interface {
	NotifyMessage(ctx context.Context, sub PushSubscription, payload PushPayload) (int, error)
}

type WebPushNotifier struct {
	publicKey  string
	privateKey string
	subject    string
	baseURL    string
	client     *http.Client
}

func NewWebPushNotifierFromEnv() *WebPushNotifier {
	return &WebPushNotifier{
		publicKey:  strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY")),
		privateKey: strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY")),
		subject:    strings.TrimSpace(os.Getenv("VAPID_SUBJECT")),
		baseURL:    strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (n *WebPushNotifier) enabled() bool {
	return n.publicKey != "" && n.privateKey != "" && n.subject != ""
}

func (n *WebPushNotifier) NotifyMessage(ctx context.Context, sub PushSubscription, payload PushPayload) (int, error) {
	if !n.enabled() {
		return 0, nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	aud, err := audienceFromEndpoint(sub.Endpoint)
	if err != nil {
		return 0, err
	}
	jwt, err := vapidJWT(n.privateKey, n.subject, aud, time.Now().Add(12*time.Hour))
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("TTL", "30")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "vapid t="+jwt+", k="+n.publicKey)
	req.Header.Set("Urgency", "high")

	resp, err := n.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		time.Sleep(200 * time.Millisecond)
		resp2, err2 := n.client.Do(req)
		if err2 == nil {
			defer resp2.Body.Close()
			return resp2.StatusCode, nil
		}
	}

	return resp.StatusCode, nil
}

func audienceFromEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("endpoint push inválido")
	}
	return u.Scheme + "://" + u.Host, nil
}

func vapidJWT(privateKeyB64, subject, aud string, exp time.Time) (string, error) {
	head := map[string]any{"typ": "JWT", "alg": "ES256"}
	claims := map[string]any{"sub": subject, "aud": aud, "exp": exp.Unix()}
	hb, _ := json.Marshal(head)
	cb, _ := json.Marshal(claims)
	unsigned := b64(hb) + "." + b64(cb)

	key, err := parseP256PrivateKey(privateKeyB64)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(unsigned))
	r, s, err := ecdsa.Sign(rand.Reader, key, h[:])
	if err != nil {
		return "", err
	}
	sig := make([]byte, 64)
	rb, sb := r.Bytes(), s.Bytes()
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):], sb)
	return unsigned + "." + b64(sig), nil
}

func parseP256PrivateKey(b64key string) (*ecdsa.PrivateKey, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(b64key))
	if err != nil {
		return nil, fmt.Errorf("VAPID_PRIVATE_KEY inválida: %w", err)
	}
	if len(raw) != 32 {
		return nil, errors.New("VAPID_PRIVATE_KEY deve ter 32 bytes")
	}
	curve := elliptic.P256()
	d := new(big.Int).SetBytes(raw)
	x, y := curve.ScalarBaseMult(raw)
	return &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y}, D: d}, nil
}

func b64(v []byte) string { return base64.RawURLEncoding.EncodeToString(v) }
