package helpers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

const csrfCookieName = "csrf_token"

func EnsureCSRFToken(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token
		}
	}

	token := randomToken(32)
	if token == "" {
		return ""
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   strings.EqualFold(os.Getenv("APP_ENV"), "prod"),
		SameSite: http.SameSiteLaxMode,
	})

	return token
}

func ValidateCSRFToken(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return false
	}

	formToken := strings.TrimSpace(r.FormValue("csrf_token"))
	cookieToken := strings.TrimSpace(cookie.Value)
	if formToken == "" || cookieToken == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(formToken), []byte(cookieToken)) == 1
}

func randomToken(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
