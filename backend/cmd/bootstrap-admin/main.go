package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	email, password, name, err := bootstrapCredentials()
	if err != nil {
		return err
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin initial administrator bootstrap: %w", err)
	}
	defer tx.Rollback(ctx)
	var lockResult string
	if err := tx.QueryRow(ctx, "SELECT pg_advisory_xact_lock(913509874)::text").Scan(&lockResult); err != nil {
		return fmt.Errorf("lock initial administrator bootstrap: %w", err)
	}
	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users WHERE role = 'admin')").Scan(&exists); err != nil {
		return fmt.Errorf("check existing administrators: %w", err)
	}
	if exists {
		return errors.New("an administrator already exists; no changes were made")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash initial administrator password: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, 'admin')
	`, name, email, string(passwordHash)); err != nil {
		return fmt.Errorf("create initial administrator: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit initial administrator: %w", err)
	}
	log.Print("Initial administrator created. Remove the bootstrap credentials from the host environment.")
	return nil
}

func bootstrapCredentials() (email, password, name string, err error) {
	email = strings.ToLower(strings.TrimSpace(os.Getenv("INITIAL_ADMIN_EMAIL")))
	password = os.Getenv("INITIAL_ADMIN_PASSWORD")
	name = strings.TrimSpace(os.Getenv("INITIAL_ADMIN_NAME"))
	parsed, parseErr := mail.ParseAddress(email)
	if parseErr != nil || parsed.Address != email {
		return "", "", "", errors.New("INITIAL_ADMIN_EMAIL must be a valid email address")
	}
	if len(password) < 12 || len(password) > 72 {
		return "", "", "", errors.New("INITIAL_ADMIN_PASSWORD must contain between 12 and 72 bytes")
	}
	if name == "" || len(name) > 120 {
		return "", "", "", errors.New("INITIAL_ADMIN_NAME must contain between 1 and 120 characters")
	}
	return email, password, name, nil
}
