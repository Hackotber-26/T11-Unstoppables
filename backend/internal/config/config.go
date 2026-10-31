package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env         string
	Host        string
	Port        int
	APIKey      string
	JWTSecret   string
	DatabaseURL string
	RedisURL    string
	CORSOrigins []string
}

func Load() *Config {
	port, err := strconv.Atoi(getEnv("APP_PORT", "8080"))
	if err != nil {
		port = 8080
	}
	cfg := &Config{
		Env:         getEnv("APP_ENV", "development"),
		Host:        getEnv("APP_HOST", "0.0.0.0"),
		Port:        port,
		APIKey:      getEnv("APP_API_KEY", ""),
		JWTSecret:   getEnv("JWT_SECRET", "local-development-only-secret-change-before-deploy-2026"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/trainsync"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		CORSOrigins: splitCSV(getEnv("APP_CORS_ORIGINS", "http://localhost:8000,http://127.0.0.1:8000")),
	}
	return cfg
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("APP_PORT must be between 1 and 65535")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.Env == "production" && c.JWTSecret == "local-development-only-secret-change-before-deploy-2026" {
		return fmt.Errorf("JWT_SECRET must be explicitly set in production")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
