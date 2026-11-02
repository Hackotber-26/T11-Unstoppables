package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"t11-unstoppables/internal/config"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type demoAccount struct {
	name  string
	email string
	role  string
}

var demoAccounts = []demoAccount{
	{name: "Demo Rider", email: "rider.demo@nimma.local", role: "rider"},
	{name: "Demo Driver", email: "driver.demo@nimma.local", role: "driver"},
	{name: "Demo Administrator", email: "admin.demo@nimma.local", role: "admin"},
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	password := os.Getenv("DEMO_PASSWORD")
	if len(password) < 12 || len(password) > 72 {
		return errors.New("DEMO_PASSWORD must contain between 12 and 72 characters")
	}
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close(ctx)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin demo seeding: %w", err)
	}
	defer tx.Rollback(ctx)

	ids := make(map[string]string, len(demoAccounts))
	for _, account := range demoAccounts {
		id, err := ensureAccount(ctx, tx, account, string(hash))
		if err != nil {
			return err
		}
		ids[account.role] = id
	}
	if err := seedRiderData(ctx, tx, ids["rider"]); err != nil {
		return err
	}
	if err := seedDriverData(ctx, tx, ids["driver"]); err != nil {
		return err
	}
	if err := seedTrips(ctx, tx, ids["rider"], ids["driver"]); err != nil {
		return err
	}
	if err := seedBays(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit demo data: %w", err)
	}
	log.Printf("demo data ready: %d accounts, stations, trips, and wallet funds", len(demoAccounts))
	return nil
}

func seedBays(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO station_bays (station_id, label, capacity, queue_length, walking_meters, accessible, is_open)
		VALUES
			('majestic', 'Demo Pickup A', 4, 0, 40, TRUE, TRUE),
			('majestic', 'Demo Pickup B', 3, 1, 85, FALSE, TRUE),
			('indiranagar', 'Demo Pickup A', 4, 0, 35, TRUE, TRUE)
		ON CONFLICT (station_id, label) DO UPDATE SET
			capacity = EXCLUDED.capacity,
			queue_length = EXCLUDED.queue_length,
			walking_meters = EXCLUDED.walking_meters,
			accessible = EXCLUDED.accessible,
			is_open = EXCLUDED.is_open
	`)
	if err != nil {
		return fmt.Errorf("seed demo station bays: %w", err)
	}
	return nil
}

func ensureAccount(ctx context.Context, tx pgx.Tx, account demoAccount, passwordHash string) (string, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE LOWER(email) = LOWER($2))
	`, account.name, account.email, passwordHash, account.role)
	if err != nil {
		return "", fmt.Errorf("create demo %s account: %w", account.role, err)
	}
	var id, role string
	err = tx.QueryRow(ctx, "SELECT id::text, role FROM users WHERE LOWER(email) = LOWER($1)", account.email).Scan(&id, &role)
	if err != nil {
		return "", fmt.Errorf("load demo %s account: %w", account.role, err)
	}
	if role != account.role {
		return "", fmt.Errorf("demo email %s already belongs to role %s", account.email, role)
	}
	return id, nil
}

func seedRiderData(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, "INSERT INTO rider_profiles (user_id, phone, home_station_id) VALUES ($1, '', 'majestic') ON CONFLICT (user_id) DO NOTHING", userID); err != nil {
		return fmt.Errorf("create demo rider profile: %w", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO wallet_accounts (user_id, currency) VALUES ($1, 'INR') ON CONFLICT (user_id) DO NOTHING", userID); err != nil {
		return fmt.Errorf("create demo rider wallet: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO wallet_ledger_entries (user_id, idempotency_key, amount_paise, entry_type, description)
		VALUES ($1, 'demo-opening-credit-v1', 100000, 'credit', 'Demo wallet opening credit')
		ON CONFLICT (idempotency_key) DO NOTHING
	`, userID); err != nil {
		return fmt.Errorf("fund demo rider wallet: %w", err)
	}
	return nil
}

func seedDriverData(ctx context.Context, tx pgx.Tx, userID string) error {
	var vehicleID string
	err := tx.QueryRow(ctx, `
		INSERT INTO fleet_vehicles (label, vehicle_type, capacity, active, verified, accessibility_supported)
		VALUES ('DEMO-AUTO-001', 'auto', 3, TRUE, TRUE, FALSE)
		ON CONFLICT (label) DO UPDATE SET active = TRUE, verified = TRUE
		RETURNING id::text
	`).Scan(&vehicleID)
	if err != nil {
		return fmt.Errorf("create demo vehicle: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO driver_profiles (user_id, vehicle_id, status, verified)
		VALUES ($1, $2, 'available', TRUE)
		ON CONFLICT (user_id) DO UPDATE
		SET vehicle_id = COALESCE(driver_profiles.vehicle_id, EXCLUDED.vehicle_id),
		    verified = TRUE
	`, userID, vehicleID); err != nil {
		return fmt.Errorf("create demo driver profile: %w", err)
	}
	return nil
}

func seedTrips(ctx context.Context, tx pgx.Tx, riderID, driverID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO trip_requests
			(id, rider_id, driver_id, origin_id, destination, distance_meters, status, pool_mode, fare_paise)
		VALUES
			('00000000-0000-4000-8000-000000000101', $1, NULL, 'majestic', 'Indiranagar', 5500, 'requested', 'corridor', 8160),
			('00000000-0000-4000-8000-000000000102', $1, $2, 'majestic', 'Indiranagar', 4100, 'matched', 'corridor', 6732),
			('00000000-0000-4000-8000-000000000103', $1, $2, 'indiranagar', 'Domlur', 2800, 'completed', 'solo', 6360)
		ON CONFLICT (id) DO NOTHING
	`, riderID, driverID)
	if err != nil {
		return fmt.Errorf("create demo trips: %w", err)
	}
	return nil
}
