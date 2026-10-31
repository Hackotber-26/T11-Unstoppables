package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"t11-unstoppables/internal/storage"
	"t11-unstoppables/queries"
)

func TestEmbeddedQueriesPrepareAgainstPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable migrated PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := storage.NewPostgresStore(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tx, err := store.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	names, err := queries.Names()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name == "create_migration_ledger.sql" {
			if _, err := tx.Exec(ctx, queries.Must(name)); err != nil {
				t.Errorf("execute %s: %v", name, err)
			}
			continue
		}
		preparedName := "query_" + strings.TrimSuffix(name, ".sql")
		if _, err := tx.Prepare(ctx, preparedName, queries.Must(name)); err != nil {
			t.Errorf("prepare %s: %v", name, err)
		}
	}
}
