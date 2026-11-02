package main

import (
	"fmt"
	"t11-unstoppables/internal/config"
)

func main() {
	cfg := config.Load()
	fmt.Printf("simulator started with env=%s\n", cfg.Env)
	fmt.Printf("Database URL configured: %t\n", cfg.DatabaseURL != "")
}
