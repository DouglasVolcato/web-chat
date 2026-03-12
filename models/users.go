package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Email     string       `json:"email"`
	Password  string       `json:"password"`
	DeletedAt sql.NullTime `json:"deleted_at"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (u *User) Create(ctx context.Context, tx *sql.Tx) error {
	u.ID = uuid.NewString()

	query := `
        insert into users (
            id,
            name,
            email,
            password
        ) values (
            $1,
            $2,
            $3,
            $4
        )
    `

	_, err := ExecContext(
		tx,
		ctx,
		query,
		u.ID,
		u.Name,
		u.Email,
		u.Password,
	)

	return err
}

func (u *User) Update(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            users
        set
            name = $1,
            email = $2,
            password = $3
        where
            id = $4
            and deleted_at is null
    `

	_, err := ExecContext(
		tx,
		ctx,
		query,
		u.Name,
		u.Email,
		u.Password,
		u.ID,
	)

	return err
}

func (u *User) Delete(ctx context.Context, tx *sql.Tx) error {
	query := `
        update
            users
        set
            name = '',
            email = '',
            password = '',
            deleted_at = NOW()
        where
            id = $1
            and deleted_at is null
    `

	_, err := ExecContext(tx, ctx, query, u.ID)
	return err
}

func GetUser(ctx context.Context, tx *sql.Tx, id string) (*User, error) {
	query := `
        select
            id,
            name,
            email,
            password,
            deleted_at,
            created_at,
            updated_at
        from users
        where id = $1
            and deleted_at is null
    `

	row := QueryRowContext(tx, ctx, query, id)

	var user User
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUsers(ctx context.Context, tx *sql.Tx) ([]User, error) {
	query := `
        select
            id,
            name,
            email,
            password,
            deleted_at,
            created_at,
            updated_at
        from users
        where deleted_at is null
    `

	rows, err := QueryContext(tx, ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User

		err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.DeletedAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func GetUserByEmail(ctx context.Context, tx *sql.Tx, email string) (*User, error) {
	query := `
        select
            id,
            name,
            email,
            password,
            deleted_at,
            created_at,
            updated_at
        from users
        where deleted_at is null
          and lower(email) = lower($1)
        limit 1
    `

	row := QueryRowContext(tx, ctx, query, email)

	var user User
	err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
