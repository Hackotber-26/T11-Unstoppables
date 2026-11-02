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
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	log.Printf("dispatch worker started env=%s", cfg.Env)
	for {
		pools, poolErr := store.FormPools(ctx)
		if poolErr != nil {
			log.Printf("pool formation cycle failed: %v", poolErr)
		} else if pools > 0 {
			log.Printf("formed %d corridor pools", pools)
		}
		count, err := store.DispatchSoloTrips(ctx, 20)
		if err != nil {
			log.Printf("dispatch cycle failed: %v", err)
		} else if count > 0 {
			log.Printf("dispatched %d solo trips", count)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
