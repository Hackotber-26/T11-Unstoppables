package main

import (
	"context"
	"log"
	"os"

	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/storage"
)

func main() {
	cfg := config.Load()
	store, err := storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer store.Close()

	directory := os.Getenv("MIGRATIONS_DIR")
	if directory == "" {
		directory = "migrations"
	}
	if err := storage.ApplyMigrations(context.Background(), store.Pool, directory); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	log.Printf("migrations applied from %s", directory)
}
