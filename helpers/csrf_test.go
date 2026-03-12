package helpers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFTokenRoundTrip(t *testing.T) {
	r := httptest.NewRequest("GET", "/app/profile", nil)
	w := httptest.NewRecorder()

	token := EnsureCSRFToken(w, r)
	if strings.TrimSpace(token) == "" {
		t.Fatal("expected csrf token")
	}

	res := w.Result()
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected csrf cookie")
	}

	post := httptest.NewRequest("POST", "/app/profile/name", strings.NewReader("csrf_token="+url.QueryEscape(token)))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(cookies[0])
	if err := post.ParseForm(); err != nil {
		t.Fatalf("parse form: %v", err)
	}

	if !ValidateCSRFToken(post) {
		t.Fatal("expected csrf to be valid")
	}
}

func TestCSRFFailsOnMismatch(t *testing.T) {
	post := httptest.NewRequest("POST", "/app/profile/name", strings.NewReader("csrf_token=aaa"))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "bbb"})
	if err := post.ParseForm(); err != nil {
		t.Fatalf("parse form: %v", err)
	}

	if ValidateCSRFToken(post) {
		t.Fatal("expected csrf mismatch to fail")
	}
}
