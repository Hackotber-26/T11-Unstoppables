package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Env                         string
	Host                        string
	Port                        int
	APIKey                      string
	JWTSecret                   string
	DatabaseURL                 string
	RedisURL                    string
	CORSOrigins                 []string
	TwilioAccountSID            string
	TwilioAuthToken             string
	TwilioVerifyServiceSID      string
	DocumentsPath               string
	DocumentsEncryptionKey      []byte
	OSRMBaseURL                 string
	GoogleMapsAPIKey            string
	documentsEncryptionKeyError error
}

func Load() *Config {
	portValue := getEnv("APP_PORT", getEnv("PORT", "8080"))
	port, err := strconv.Atoi(portValue)
	if err != nil {
		port = 8080
	}
	encryptionKeyValue := getEnv("DRIVER_DOCUMENTS_ENCRYPTION_KEY", "")
	encryptionKey, encryptionKeyErr := base64.StdEncoding.DecodeString(encryptionKeyValue)
	if encryptionKeyValue == "" {
		encryptionKey = nil
		encryptionKeyErr = nil
	}
	cfg := &Config{
		Env:                         getEnv("APP_ENV", "development"),
		Host:                        getEnv("APP_HOST", "0.0.0.0"),
		Port:                        port,
		APIKey:                      getEnv("APP_API_KEY", ""),
		JWTSecret:                   getEnv("JWT_SECRET", "local-development-only-secret-change-before-deploy-2026"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/trainsync"),
		RedisURL:                    getEnv("REDIS_URL", ""),
		CORSOrigins:                 splitCSV(getEnv("APP_CORS_ORIGINS", "http://localhost:8000,http://127.0.0.1:8000")),
		TwilioAccountSID:            getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:             getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioVerifyServiceSID:      getEnv("TWILIO_VERIFY_SERVICE_SID", ""),
		DocumentsPath:               getEnv("DRIVER_DOCUMENTS_PATH", "private/driver-documents"),
		DocumentsEncryptionKey:      encryptionKey,
		OSRMBaseURL:                 getEnv("OSRM_BASE_URL", ""),
		GoogleMapsAPIKey:            getEnv("GOOGLE_MAPS_API_KEY", ""),
		documentsEncryptionKeyError: encryptionKeyErr,
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
	if c.documentsEncryptionKeyError != nil {
		return fmt.Errorf("DRIVER_DOCUMENTS_ENCRYPTION_KEY must be valid base64")
	}
	if len(c.DocumentsEncryptionKey) != 0 && len(c.DocumentsEncryptionKey) != 32 {
		return fmt.Errorf("DRIVER_DOCUMENTS_ENCRYPTION_KEY must decode to exactly 32 bytes")
	}
	if c.Env == "production" {
		if len(c.DocumentsEncryptionKey) != 32 {
			return fmt.Errorf("DRIVER_DOCUMENTS_ENCRYPTION_KEY must decode to exactly 32 bytes in production")
		}
		databaseURL, err := url.Parse(c.DatabaseURL)
		if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") || databaseURL.Host == "" {
			return fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with TLS enabled in production")
		}
		switch databaseURL.Query().Get("sslmode") {
		case "require", "verify-ca", "verify-full":
		default:
			return fmt.Errorf("DATABASE_URL must set sslmode=require or stronger in production")
		}
		if len(c.CORSOrigins) == 0 {
			return fmt.Errorf("APP_CORS_ORIGINS must contain at least one trusted HTTPS origin in production")
		}
		for _, origin := range c.CORSOrigins {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.Contains(parsed.Host, "*") || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
				return fmt.Errorf("APP_CORS_ORIGINS must contain exact HTTPS origins without paths or query strings in production")
			}
		}
		routerURL, err := url.Parse(c.OSRMBaseURL)
		if err != nil || routerURL.Scheme != "https" || routerURL.Host == "" || routerURL.User != nil || routerURL.RawQuery != "" || routerURL.Fragment != "" {
			return fmt.Errorf("OSRM_BASE_URL must be an HTTPS URL without query or fragment in production")
		}
		if strings.TrimSpace(c.GoogleMapsAPIKey) == "" {
			return fmt.Errorf("GOOGLE_MAPS_API_KEY is required in production for transit routing")
		}
		if !filepath.IsAbs(c.DocumentsPath) {
			return fmt.Errorf("DRIVER_DOCUMENTS_PATH must be an absolute persistent path in production")
		}
	}
	twilioConfigured := c.TwilioAccountSID != "" || c.TwilioAuthToken != "" || c.TwilioVerifyServiceSID != ""
	if twilioConfigured && (c.TwilioAccountSID == "" || c.TwilioAuthToken == "" || c.TwilioVerifyServiceSID == "") {
		return fmt.Errorf("TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, and TWILIO_VERIFY_SERVICE_SID must be configured together")
	}
	if c.DocumentsPath == "" {
		return fmt.Errorf("DRIVER_DOCUMENTS_PATH is required")
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
