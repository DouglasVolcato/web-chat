package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultMigrationsDir = "migrations"

func RunMigrations(ctx context.Context) error {
	dir := defaultMigrationsDir

	if err := ensureMigrationsTable(ctx); err != nil {
		return err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, entry := range files {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}

	sort.Strings(migrationFiles)

	for _, name := range migrationFiles {
		applied, err := migrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if err := applyMigration(ctx, name, string(content)); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context) error {
	query := `
        create table if not exists schema_migrations (
            version text primary key,
            applied_at timestamptz not null default now()
        )
    `
	_, err := ExecContext(nil, ctx, query)
	return err
}

func migrationApplied(ctx context.Context, version string) (bool, error) {
	query := `select exists(select 1 from schema_migrations where version = $1)`
	row := QueryRowContext(nil, ctx, query, version)

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func applyMigration(ctx context.Context, version string, sqlContent string) error {
	if strings.TrimSpace(sqlContent) == "" {
		return fmt.Errorf("migration %s is empty", version)
	}

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, sqlContent); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply migration %s: %w", version, err)
	}

	if _, err := tx.ExecContext(ctx, `insert into schema_migrations (version) values ($1)`, version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	time.Sleep(10 * time.Millisecond)
	return nil
}
