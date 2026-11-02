package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/storage"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	store, err := storage.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer store.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	log.Printf("pool formation worker started env=%s", cfg.Env)
	for {
		count, err := store.FormPools(ctx)
		if err != nil {
			log.Printf("pool formation cycle failed: %v", err)
		} else if count > 0 {
			log.Printf("formed %d rider pools", count)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
