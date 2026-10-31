package main

import (
	"log"
	"t11-unstoppables/internal/api"
	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/storage"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	client, err := storage.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}
	defer client.Close()

	r, err := api.NewRouter(cfg, client)
	if err != nil {
		log.Fatalf("failed to create router: %v", err)
	}
	log.Printf("TrainSync API listening on %s", cfg.Addr())
	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
