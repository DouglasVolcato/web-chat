package profile

import (
	"context"
	"database/sql"

	"app/models"
)

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

func (r *PostgresRepository) GetByID(ctx context.Context, tx *sql.Tx, userID string) (*models.User, error) {
	return models.GetUser(ctx, tx, userID)
}

func (r *PostgresRepository) UpdateName(ctx context.Context, tx *sql.Tx, userID, name string) error {
	query := `
		update users
		set
			name = $1,
			updated_at = now()
		where id = $2
		  and deleted_at is null
	`

	_, err := models.ExecContext(tx, ctx, query, name, userID)
	return err
}

func (r *PostgresRepository) DeleteUserChats(ctx context.Context, tx *sql.Tx, userID string) error {
	query := `
		delete from user_chats
		where user_id = $1
	`

	_, err := models.ExecContext(tx, ctx, query, userID)
	return err
}

func (r *PostgresRepository) SoftDeleteAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	query := `
		update users
		set
			name = 'Deleted User',
			email = concat('deleted+', id::text, '@invalid.local'),
			password = '',
			deleted_at = now(),
			updated_at = now()
		where id = $1
		  and deleted_at is null
	`

	_, err := models.ExecContext(tx, ctx, query, userID)
	return err
}
