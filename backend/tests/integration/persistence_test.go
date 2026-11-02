package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/services"
	"t11-unstoppables/internal/storage"

	"github.com/google/uuid"
)

func TestPersistentRiderTripAndWalletLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable migrated PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	store, err := storage.NewPostgresStore(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, file, _, _ := runtime.Caller(0)
	migrations := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	if err := storage.ApplyMigrations(ctx, store.Pool, migrations); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	email := fmt.Sprintf("integration-%s@example.test", uuid.NewString())
	user, err := store.CreateRider(ctx, "Integration Rider", email, "", "test-bcrypt-hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateRider(ctx, "Duplicate Rider", email, "", "test-bcrypt-hash"); err != storage.ErrConflict {
		t.Fatalf("duplicate email error = %v, want %v", err, storage.ErrConflict)
	}
	userIDs := []string{user.ID}
	t.Cleanup(func() {
		cleanup, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		for _, userID := range userIDs {
			_, _ = store.Pool.Exec(cleanup, "DELETE FROM wallet_ledger_entries WHERE user_id = $1", userID)
			_, _ = store.Pool.Exec(cleanup, "DELETE FROM trip_requests WHERE rider_id = $1", userID)
			_, _ = store.Pool.Exec(cleanup, "DELETE FROM audit_events WHERE actor_id = $1", userID)
			_, _ = store.Pool.Exec(cleanup, "DELETE FROM users WHERE id = $1", userID)
		}
	})

	originID := "indiranagar"
	destination := "destination-integration-" + uuid.NewString()
	trip, err := store.CreateTrip(ctx, user.ID, originID, destination, "solo", 3200, 730)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTrip(ctx, user.ID, "unknown-station", destination, "solo", 3200, 730); err != storage.ErrInvalidStation {
		t.Fatalf("unknown origin station error = %v, want %v", err, storage.ErrInvalidStation)
	}
	loadedTrip, err := store.TripByID(ctx, trip.ID)
	if err != nil || loadedTrip.RiderID != user.ID || loadedTrip.FareCents != 730 {
		t.Fatalf("trip did not round-trip: %+v, %v", loadedTrip, err)
	}
	userTrips, err := store.ListTrips(ctx, user.ID, "rider", 50)
	if err != nil || !hasTrip(userTrips, trip.ID) {
		t.Fatalf("rider trip listing omitted created trip: %v", err)
	}
	riders, err := store.ListRiders(ctx)
	if err != nil || !hasRider(riders, user.ID) {
		t.Fatalf("rider listing omitted new user: %v", err)
	}
	_, err = store.ListDrivers(ctx)
	if err != nil {
		t.Fatalf("driver listing query: %v", err)
	}
	if _, err := store.TripCounts(ctx); err != nil {
		t.Fatalf("trip metrics query: %v", err)
	}

	secondEmail := fmt.Sprintf("integration-%s@example.test", uuid.NewString())
	secondUser, err := store.CreateRider(ctx, "Pool Candidate", secondEmail, "", "test-bcrypt-hash")
	if err != nil {
		t.Fatal(err)
	}
	userIDs = append(userIDs, secondUser.ID)
	if _, err := store.CreateTrip(ctx, secondUser.ID, originID, destination, "corridor", 3200, 620); err != nil {
		t.Fatal(err)
	}
	candidates, err := store.PoolCandidates(ctx, user.ID)
	if err != nil || len(candidates) != 1 || candidates[0].RiderID != secondUser.ID {
		t.Fatalf("pool candidate query returned %+v, err=%v", candidates, err)
	}

	key := "integration-credit-" + uuid.NewString()
	balance, err := store.AppendWalletEntry(ctx, user.ID, user.ID, "", key, "credit", "integration test credit", 1000)
	if err != nil || balance != 1000 {
		t.Fatalf("credit balance = %d, err=%v", balance, err)
	}
	chargeKey := "integration-charge-" + uuid.NewString()
	balance, err = store.AppendWalletEntry(ctx, user.ID, user.ID, trip.ID, chargeKey, "debit", "integration trip charge", -trip.FareCents)
	if err != nil || balance != 270 {
		t.Fatalf("post-charge balance = %d, err=%v", balance, err)
	}
	retryBalance, err := store.AppendWalletEntry(ctx, user.ID, user.ID, trip.ID, chargeKey, "debit", "integration trip charge", -trip.FareCents)
	if err != nil || retryBalance != balance {
		t.Fatalf("idempotent retry balance = %d, err=%v", retryBalance, err)
	}
	if _, err := store.AppendWalletEntry(ctx, user.ID, user.ID, trip.ID, chargeKey, "debit", "different amount", -trip.FareCents-1); err != storage.ErrIdempotencyConflict {
		t.Fatalf("changed idempotency operation error = %v, want %v", err, storage.ErrIdempotencyConflict)
	}
	if _, err := store.AppendWalletEntry(ctx, user.ID, user.ID, "", "integration-overdraft-"+uuid.NewString(), "debit", "overdraft", -1000); err != storage.ErrInsufficientFunds {
		t.Fatalf("overdraft error = %v, want %v", err, storage.ErrInsufficientFunds)
	}
	entries, err := store.WalletEntries(ctx, user.ID, 10)
	if err != nil || len(entries) != 2 {
		t.Fatalf("wallet entry listing returned %d entries, err=%v", len(entries), err)
	}
	if _, err := store.FormPools(ctx); err != nil {
		t.Fatalf("empty pool worker query: %v", err)
	}
	if count, err := store.DispatchSoloTrips(ctx, 1); err != nil || count != 0 {
		t.Fatalf("dispatch without verified fleet = count %d, err=%v", count, err)
	}
	if _, err := store.TransitionTrip(ctx, trip.ID, user.ID, "rider", "cancelled"); err != nil {
		t.Fatalf("rider trip cancellation: %v", err)
	}
	coordinator := services.NewCoordinator(nil, &storage.Client{Postgres: store})
	if _, err := coordinator.PayTrip(ctx, user.ID, trip.ID, "cancelled-trip-payment"); !errors.Is(err, storage.ErrTripNotPayable) {
		t.Fatalf("cancelled trip payment error = %v, want %v", err, storage.ErrTripNotPayable)
	}
}

func hasRider(riders []models.Rider, userID string) bool {
	for _, rider := range riders {
		if rider.ID == userID {
			return true
		}
	}
	return false
}

func hasTrip(trips []models.Trip, tripID string) bool {
	for _, trip := range trips {
		if trip.ID == tripID {
			return true
		}
	}
	return false
}
