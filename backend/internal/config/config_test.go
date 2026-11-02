package config

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestLoadUsesPlatformPortAndLeavesRedisOptional(t *testing.T) {
	t.Setenv("APP_PORT", "")
	t.Setenv("PORT", "10000")
	t.Setenv("REDIS_URL", "")
	t.Setenv("DRIVER_DOCUMENTS_ENCRYPTION_KEY", "")

	cfg := Load()
	if cfg.Port != 10000 {
		t.Fatalf("port = %d, want platform port 10000", cfg.Port)
	}
	if cfg.RedisURL != "" {
		t.Fatalf("redis URL = %q, want disabled by default", cfg.RedisURL)
	}
}

func TestLoadDecodesDocumentEncryptionKey(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	raw := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32))
	t.Setenv("DRIVER_DOCUMENTS_ENCRYPTION_KEY", raw)
	cfg := Load()
	if len(cfg.DocumentsEncryptionKey) != 32 || cfg.documentsEncryptionKeyError != nil {
		t.Fatalf("document encryption key was not loaded correctly: bytes=%d error=%v", len(cfg.DocumentsEncryptionKey), cfg.documentsEncryptionKeyError)
	}

	t.Setenv("DRIVER_DOCUMENTS_ENCRYPTION_KEY", "not-base64")
	cfg = Load()
	if cfg.documentsEncryptionKeyError == nil {
		t.Fatal("expected invalid base64 document key to be recorded")
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected malformed encryption configuration to fail validation")
	}
}

func TestValidateRejectsWeakProductionSecret(t *testing.T) {
	cfg := &Config{
		Env: "production", Host: "0.0.0.0", Port: 8080,
		JWTSecret:              "local-development-only-secret-change-before-deploy-2026",
		DatabaseURL:            "postgres://localhost/trainsync?sslmode=require",
		DocumentsPath:          "/var/data/driver-documents",
		CORSOrigins:            []string{"https://app.example.com"},
		OSRMBaseURL:            "https://routes.example.com",
		GoogleMapsAPIKey:       "test-google-maps-api-key",
		DocumentsEncryptionKey: bytes.Repeat([]byte{1}, 32),
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production secret to be required")
	}
	cfg.JWTSecret = "production-secret-at-least-thirty-two-characters"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequiresProductionProviderConfiguration(t *testing.T) {
	cfg := &Config{
		Env: "production", Port: 8080, JWTSecret: "production-secret-at-least-thirty-two-characters",
		DatabaseURL: "postgres://localhost/trainsync?sslmode=require", DocumentsPath: "/var/data/driver-documents",
		CORSOrigins:            []string{"https://app.example.com"},
		DocumentsEncryptionKey: bytes.Repeat([]byte{1}, 32),
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing production routing configuration to be rejected")
	}
	cfg.OSRMBaseURL = "http://router.internal"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected non-HTTPS production routing URL to be rejected")
	}
	cfg.OSRMBaseURL = "https://routes.example.com"
	cfg.GoogleMapsAPIKey = "test-google-maps-api-key"
	cfg.CORSOrigins = []string{"*"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected wildcard production CORS origin to be rejected")
	}
}

func TestValidateRejectsProductionCorsPaths(t *testing.T) {
	cfg := &Config{
		Env: "production", Port: 8080, JWTSecret: "production-secret-at-least-thirty-two-characters",
		DatabaseURL: "postgres://localhost/trainsync?sslmode=require", DocumentsPath: "/var/data/driver-documents",
		CORSOrigins: []string{"https://app.example.com/path"},
		OSRMBaseURL: "https://routes.example.com", GoogleMapsAPIKey: "test-google-maps-api-key",
		DocumentsEncryptionKey: bytes.Repeat([]byte{1}, 32),
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected path-qualified CORS origins to be rejected")
	}
}
