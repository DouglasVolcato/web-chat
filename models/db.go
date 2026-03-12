package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"app/observability"
)

var counts = 0
var DB *sql.DB

func NewDB(dbPool *sql.DB) {
	DB = dbPool
}

func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5",
			os.Getenv("POSTGRES_HOST"),
			os.Getenv("POSTGRES_PORT"),
			os.Getenv("POSTGRES_USER"),
			os.Getenv("POSTGRES_PASSWORD"),
			os.Getenv("POSTGRES_DB"),
		)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func ConnectToDB() error {
	for {
		connection, err := openDB()

		if err != nil {
			log.Println("Error connectiong to Postgres", err)
			counts++
		} else {
			log.Println("Connected to Postgres")
			NewDB(connection)
			return nil
		}

		if counts > 10 {
			log.Println(err)
			return errors.New("Can't connect to Postgres")
		}

		log.Println("Backing off for two seconds")
		time.Sleep(time.Second * 2)
		continue
	}
}

func BeginTransaction(ctx context.Context, timeout time.Duration) (context.Context, *sql.Tx, func(), error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)

	if DB == nil {
		cancel()
		return nil, nil, nil, errors.New("database connection is not initialized")
	}

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}

	done := func() {
		if p := recover(); p != nil {
			tx.Rollback()
			cancel()
			panic(p)
		} else {
			if err := ctx.Err(); err != nil {
				tx.Rollback()
			} else {
				if commitErr := tx.Commit(); commitErr != nil {
					log.Println("Error committing transaction:", commitErr)
				}
			}
			cancel()
		}
	}

	return ctx, tx, done, nil
}

func ExecContext(tx *sql.Tx, ctx context.Context, query string, args ...any) (sql.Result, error) {
	if observability.DebugLoggingEnabled() {
		observability.SQLLogger().Printf("query=%s args=%v", query, args)
	}

	if tx == nil {
		return DB.ExecContext(ctx, query, args...)
	}

	return tx.ExecContext(ctx, query, args...)
}

func QueryContext(tx *sql.Tx, ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if observability.DebugLoggingEnabled() {
		observability.SQLLogger().Printf("query=%s args=%v", query, args)
	}

	if tx == nil {
		return DB.QueryContext(ctx, query, args...)
	}

	return tx.QueryContext(ctx, query, args...)
}

func QueryRowContext(tx *sql.Tx, ctx context.Context, query string, args ...any) *sql.Row {
	if observability.DebugLoggingEnabled() {
		observability.SQLLogger().Printf("query=%s args=%v", query, args)
	}

	if tx == nil {
		return DB.QueryRowContext(ctx, query, args...)
	}

	return tx.QueryRowContext(ctx, query, args...)
}
