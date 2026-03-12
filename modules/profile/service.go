package profile

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

const (
	minNameLength = 2
	maxNameLength = 60
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, tx *sql.Tx, userID string) (*ProfileDTO, *AppError) {
	if strings.TrimSpace(userID) == "" {
		return nil, newUnauthorizedError()
	}

	user, err := s.repo.GetByID(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newNotFoundError()
		}
		return nil, newInternalError(err)
	}

	return &ProfileDTO{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}

func (s *Service) UpdateName(ctx context.Context, tx *sql.Tx, userID, name string) (*ProfileDTO, *AppError) {
	if strings.TrimSpace(userID) == "" {
		return nil, newUnauthorizedError()
	}

	name = strings.TrimSpace(name)
	if len(name) < minNameLength {
		return nil, newValidationError("invalid_name", "Nome deve ter pelo menos 2 caracteres.")
	}
	if len(name) > maxNameLength {
		return nil, newValidationError("invalid_name", "Nome deve ter no máximo 60 caracteres.")
	}

	if err := s.repo.UpdateName(ctx, tx, userID, name); err != nil {
		return nil, newInternalError(err)
	}

	return s.GetProfile(ctx, tx, userID)
}

func (s *Service) DeleteAccount(ctx context.Context, tx *sql.Tx, userID string) *AppError {
	if strings.TrimSpace(userID) == "" {
		return newUnauthorizedError()
	}

	if _, err := s.repo.GetByID(ctx, tx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newNotFoundError()
		}
		return newInternalError(err)
	}

	if err := s.repo.DeleteUserChats(ctx, tx, userID); err != nil {
		return newInternalError(err)
	}

	if err := s.repo.SoftDeleteAccount(ctx, tx, userID); err != nil {
		return newInternalError(err)
	}

	return nil
}

type ProfileDTO struct {
	ID    string
	Name  string
	Email string
}
