package config

import "testing"

func TestValidateRejectsWeakProductionSecret(t *testing.T) {
	cfg := &Config{Env: "production", Host: "0.0.0.0", Port: 8080, JWTSecret: "local-development-only-secret-change-before-deploy-2026", DatabaseURL: "postgres://localhost/trainsync"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production secret to be required")
	}
	cfg.JWTSecret = "production-secret-at-least-thirty-two-characters"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}