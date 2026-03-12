package profile

import (
	"context"
	"database/sql"

	"app/models"
)

// Repository defines persistence contracts for profile operations.
type Repository interface {
	GetByID(ctx context.Context, tx *sql.Tx, userID string) (*models.User, error)
	UpdateName(ctx context.Context, tx *sql.Tx, userID, name string) error
	DeleteUserChats(ctx context.Context, tx *sql.Tx, userID string) error
	SoftDeleteAccount(ctx context.Context, tx *sql.Tx, userID string) error
}
