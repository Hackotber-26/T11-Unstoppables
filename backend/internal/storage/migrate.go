package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"t11-unstoppables/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, directory string) error {
	if pool == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if _, err := pool.Exec(ctx, queries.Must("create_migration_ledger.sql")); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(directory, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)
	for _, file := range files {
		version := filepath.Base(file)
		var exists bool
		if err := pool.QueryRow(ctx, queries.Must("migration_exists.sql"), version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", version, err)
		}
		for _, statement := range strings.Split(string(contents), ";") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := tx.Exec(ctx, statement); err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("execute migration %s: %w", version, err)
			}
		}
		if _, err := tx.Exec(ctx, queries.Must("record_migration.sql"), version); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", version, err)
		}
	}
	return nil
}
