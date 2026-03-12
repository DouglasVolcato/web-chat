package helpers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/idtoken"
)

// GoogleProfile representa os dados básicos extraídos do token do Google.
type GoogleProfile struct {
	Email string
	Name  string
}

// VerifyGoogleIDToken valida o token do Google e retorna e-mail e nome do usuário.
func VerifyGoogleIDToken(ctx context.Context, token string) (*GoogleProfile, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID não configurado")
	}

	payload, err := idtoken.Validate(ctx, token, clientID)
	if err != nil {
		return nil, fmt.Errorf("token do Google inválido: %w", err)
	}

	email := ""
	if claimEmail, ok := payload.Claims["email"].(string); ok {
		email = strings.TrimSpace(claimEmail)
	}

	if email == "" {
		return nil, errors.New("e-mail do Google não encontrado")
	}

	emailVerified := false
	switch v := payload.Claims["email_verified"].(type) {
	case bool:
		emailVerified = v
	case string:
		emailVerified = strings.EqualFold(v, "true") || v == "1"
	}

	if !emailVerified {
		return nil, errors.New("e-mail do Google não verificado")
	}

	name := ""
	if claimName, ok := payload.Claims["name"].(string); ok {
		name = strings.TrimSpace(claimName)
	}

	if name == "" {
		if given, ok := payload.Claims["given_name"].(string); ok {
			name = strings.TrimSpace(given)
		}
	}

	return &GoogleProfile{
		Email: strings.ToLower(email),
		Name:  name,
	}, nil
}
