package helpers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"app/models"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	userContextKey contextKey = "auth_user_id"
	authCookieName            = "auth_token"
)

// AuthClaims representa o payload do token JWT usado para autenticação.
type AuthClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// AuthDecorator garante que a requisição contenha um token JWT válido no cookie ou no header Authorization.
// Ele é usado para rotas protegidas do app e retorna 401 quando o token está ausente ou inválido.
func AuthDecorator(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := ResolveUserIDFromRequest(r)
		if err != nil || strings.TrimSpace(userID) == "" {
			RenderUnauthorized(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, strings.TrimSpace(userID))
		next(w, r.WithContext(ctx))
	}
}

// GetAuthUser recupera o usuário autenticado a partir do token JWT ou do contexto.
// Ele valida a existência no banco utilizando a transação atual.
func GetAuthUser(ctx context.Context, tx *sql.Tx, r *http.Request) (*models.User, error) {
	userID := ""

	if ctxValue := r.Context().Value(userContextKey); ctxValue != nil {
		userID = fmt.Sprint(ctxValue)
	}

	if userID == "" {
		var err error
		userID, err = ResolveUserIDFromRequest(r)
		if err != nil {
			userID = strings.TrimSpace(r.URL.Query().Get("user_id"))
		}
	}

	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("usuário não autenticado")
	}

	user, err := models.GetUser(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("usuário não encontrado")
		}
		return nil, err
	}

	return user, nil
}

// SetAuthCookie gera um token JWT e envia um cookie HttpOnly para autenticação automática pelo navegador.
func SetAuthCookie(w http.ResponseWriter, userID string, duration time.Duration) error {
	token, err := generateJWT(userID, duration)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	if duration > 0 {
		cookie.Expires = time.Now().Add(duration)
		cookie.MaxAge = int(duration.Seconds())
	}

	http.SetCookie(w, cookie)
	return nil
}

// ClearAuthCookie remove o cookie de autenticação.
func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// RenderUnauthorized responde com uma página simples de não autorizado.
func RenderUnauthorized(w http.ResponseWriter, r *http.Request) {
	ClearAuthCookie(w)
	Redirect(w, r, "/login")
	return
}

func ResolveUserIDFromRequest(r *http.Request) (string, error) {
	if ctxValue := r.Context().Value(userContextKey); ctxValue != nil {
		if id := strings.TrimSpace(fmt.Sprint(ctxValue)); id != "" {
			return id, nil
		}
	}

	if cookie, err := r.Cookie(authCookieName); err == nil {
		if id, err := parseUserIDFromToken(cookie.Value); err == nil {
			return id, nil
		}
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
		return parseUserIDFromToken(token)
	}

	return "", errors.New("token ausente ou inválido")
}

func parseUserIDFromToken(tokenString string) (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", errors.New("jwt secret não configurado")
	}

	claims := &AuthClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("assinatura inesperada: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", errors.New("token inválido")
	}

	if claims.Subject != "" {
		return claims.Subject, nil
	}

	if claims.UserID != "" {
		return claims.UserID, nil
	}

	return "", errors.New("token sem identificador de usuário")
}

func generateJWT(userID string, duration time.Duration) (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", errors.New("jwt secret não configurado")
	}

	now := time.Now()
	claims := AuthClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
