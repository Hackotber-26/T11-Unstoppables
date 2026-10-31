package storage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	auditdomain "t11-unstoppables/internal/audit"
	"t11-unstoppables/internal/matching"
	"t11-unstoppables/internal/models"
	tripdomain "t11-unstoppables/internal/trips"
	"t11-unstoppables/queries"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record already exists")
var ErrInsufficientFunds = errors.New("insufficient wallet funds")
var ErrIdempotencyConflict = errors.New("idempotency key was already used for a different operation")
var ErrTripNotPayable = errors.New("trip is not payable in its current state")

func (s *PostgresStore) CreateRider(ctx context.Context, name, email, phone, passwordHash string) (models.User, error) {
	if s == nil || s.Pool == nil {
		return models.User{}, errors.New("postgres is not configured")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("begin rider registration: %w", err)
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	var user models.User
	err = tx.QueryRow(ctx, queries.Must("create_rider.sql"),
		strings.TrimSpace(name), strings.TrimSpace(email), passwordHash,
	).Scan(&id, &user.Name, &user.Email, &user.Role, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, ErrConflict
		}
		return models.User{}, fmt.Errorf("insert rider account: %w", err)
	}
	user.ID = id.String()
	if _, err := tx.Exec(ctx, queries.Must("create_rider_profile.sql"), id, strings.TrimSpace(phone)); err != nil {
		return models.User{}, fmt.Errorf("insert rider profile: %w", err)
	}
	if _, err := tx.Exec(ctx, queries.Must("create_wallet_account.sql"), id); err != nil {
		return models.User{}, fmt.Errorf("create rider wallet: %w", err)
	}
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: id.String(), Action: "user.registered",
		Resource: "user", ResourceID: id.String(), OccurredAt: time.Now().UTC(),
	}); err != nil {
		return models.User{}, fmt.Errorf("audit rider registration: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.User{}, fmt.Errorf("commit rider registration: %w", err)
	}
	return user, nil
}

func (s *PostgresStore) UserByEmail(ctx context.Context, email string) (models.User, error) {
	if s == nil || s.Pool == nil {
		return models.User{}, errors.New("postgres is not configured")
	}
	var id uuid.UUID
	var user models.User
	err := s.Pool.QueryRow(ctx, queries.Must("find_user_by_email.sql"), strings.TrimSpace(email)).Scan(&id, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("find user by email: %w", err)
	}
	user.ID = id.String()
	return user, nil
}

func (s *PostgresStore) ListRiders(ctx context.Context) ([]models.Rider, error) {
	rows, err := s.Pool.Query(ctx, queries.Must("list_riders.sql"))
	if err != nil {
		return nil, fmt.Errorf("list riders: %w", err)
	}
	defer rows.Close()
	result := make([]models.Rider, 0)
	for rows.Next() {
		var id uuid.UUID
		var rider models.Rider
		if err := rows.Scan(&id, &rider.Name, &rider.Email, &rider.Phone, &rider.StationID, &rider.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rider: %w", err)
		}
		rider.ID = id.String()
		result = append(result, rider)
	}
	return result, rows.Err()
}

func (s *PostgresStore) ListDrivers(ctx context.Context) ([]models.Driver, error) {
	rows, err := s.Pool.Query(ctx, queries.Must("list_drivers.sql"))
	if err != nil {
		return nil, fmt.Errorf("list drivers: %w", err)
	}
	defer rows.Close()
	result := make([]models.Driver, 0)
	for rows.Next() {
		var id uuid.UUID
		var driver models.Driver
		if err := rows.Scan(&id, &driver.Name, &driver.VehicleID, &driver.Status, &driver.Capacity); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		driver.ID = id.String()
		result = append(result, driver)
	}
	return result, rows.Err()
}

func (s *PostgresStore) CreateTrip(ctx context.Context, riderID, originID, destination, poolMode string, distanceMeters float64, fareCents int64) (models.Trip, error) {
	id, err := uuid.Parse(riderID)
	if err != nil {
		return models.Trip{}, fmt.Errorf("invalid rider id: %w", err)
	}
	var tripID uuid.UUID
	var riderUUID uuid.UUID
	var trip models.Trip
	err = s.Pool.QueryRow(ctx, queries.Must("create_trip.sql"),
		id, strings.TrimSpace(originID), strings.TrimSpace(destination), distanceMeters, poolMode, fareCents,
	).Scan(&tripID, &riderUUID, &trip.DriverID, &trip.OriginID, &trip.Destination,
		&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents, &trip.CreatedAt, &trip.UpdatedAt)
	if err != nil {
		return models.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	trip.ID = tripID.String()
	trip.RiderID = riderUUID.String()
	trip.FareAmount = float64(trip.FareCents) / 100
	return trip, nil
}

func (s *PostgresStore) TripByID(ctx context.Context, tripID string) (models.Trip, error) {
	id, err := uuid.Parse(tripID)
	if err != nil {
		return models.Trip{}, ErrNotFound
	}
	var tripIDValue uuid.UUID
	var riderUUID uuid.UUID
	var trip models.Trip
	err = s.Pool.QueryRow(ctx, queries.Must("find_trip_by_id.sql"), id).Scan(&tripIDValue, &riderUUID, &trip.DriverID, &trip.OriginID, &trip.Destination,
		&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents, &trip.CreatedAt, &trip.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Trip{}, ErrNotFound
	}
	if err != nil {
		return models.Trip{}, fmt.Errorf("find trip: %w", err)
	}
	trip.ID = tripIDValue.String()
	trip.RiderID = riderUUID.String()
	trip.FareAmount = float64(trip.FareCents) / 100
	return trip, nil
}

func (s *PostgresStore) ListTrips(ctx context.Context, userID, role string, limit int) ([]models.Trip, error) {
	if s == nil || s.Pool == nil {
		return nil, errors.New("postgres is not configured")
	}
	if role != "rider" && role != "driver" && role != "admin" {
		return nil, errors.New("unsupported trip listing role")
	}
	var actorID interface{}
	if role != "admin" {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return nil, ErrNotFound
		}
		actorID = parsed
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.Pool.Query(ctx, queries.Must("list_trips_by_actor.sql"), role, actorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list trips: %w", err)
	}
	defer rows.Close()
	result := make([]models.Trip, 0)
	for rows.Next() {
		var tripID, riderID uuid.UUID
		var trip models.Trip
		if err := rows.Scan(&tripID, &riderID, &trip.DriverID, &trip.OriginID, &trip.Destination,
			&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents, &trip.CreatedAt, &trip.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan trip: %w", err)
		}
		trip.ID = tripID.String()
		trip.RiderID = riderID.String()
		trip.FareAmount = float64(trip.FareCents) / 100
		result = append(result, trip)
	}
	return result, rows.Err()
}

func (s *PostgresStore) PoolCandidates(ctx context.Context, riderID string) ([]models.PoolCandidate, error) {
	tripID, err := uuid.Parse(riderID)
	if err != nil {
		return nil, fmt.Errorf("invalid rider id: %w", err)
	}
	rows, err := s.Pool.Query(ctx, queries.Must("find_pool_candidates.sql"), tripID)
	if err != nil {
		return nil, fmt.Errorf("find pool candidates: %w", err)
	}
	defer rows.Close()
	result := make([]models.PoolCandidate, 0)
	for rows.Next() {
		var candidateRiderID uuid.UUID
		var destination, currentDestination string
		var distance, currentDistance float64
		var createdAt, currentCreatedAt time.Time
		if err := rows.Scan(&candidateRiderID, &destination, &distance, &createdAt, &currentDestination, &currentDistance, &currentCreatedAt); err != nil {
			return nil, fmt.Errorf("scan pool candidate: %w", err)
		}
		pickupDelta := int(math.Abs(createdAt.Sub(currentCreatedAt).Seconds()))
		corridorSimilarity := 0.6
		if strings.EqualFold(strings.TrimSpace(destination), strings.TrimSpace(currentDestination)) {
			corridorSimilarity = 1
		}
		match, err := matching.Score(matching.Request{
			PartySize: 1, MaxWaitSeconds: 300, MaxDetourMeters: 2500, MaxPickupDeltaSecs: 300,
		}, matching.Candidate{
			ID: candidateRiderID.String(), AvailableSeats: 1,
			WaitSeconds:        int(time.Since(createdAt).Seconds()),
			DetourMeters:       math.Abs(distance - currentDistance),
			PickupDeltaSeconds: pickupDelta, CorridorSimilarity: corridorSimilarity,
			SafetyCompatible: true,
		})
		if err != nil {
			return nil, fmt.Errorf("score pool candidate: %w", err)
		}
		if match.Eligible {
			reason := "same origin and compatible pickup window"
			if corridorSimilarity == 1 {
				reason = "same origin, destination, and pickup window"
			}
			result = append(result, models.PoolCandidate{RiderID: candidateRiderID.String(), Score: match.Score, Reason: reason})
		}
	}
	return result, rows.Err()
}

func (s *PostgresStore) WalletBalance(ctx context.Context, userID string) (int64, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user id: %w", err)
	}
	var balance int64
	err = s.Pool.QueryRow(ctx, queries.Must("read_wallet_balance.sql"), id).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("read wallet balance: %w", err)
	}
	return balance, nil
}

func (s *PostgresStore) AppendWalletEntry(ctx context.Context, actorID, userID, tripID, key, entryType, description string, amountCents int64) (int64, error) {
	if s == nil || s.Pool == nil {
		return 0, errors.New("postgres is not configured")
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user id: %w", err)
	}
	actorUUID, err := uuid.Parse(actorID)
	if err != nil {
		return 0, fmt.Errorf("invalid actor id: %w", err)
	}
	if key == "" || len(key) > 160 || description == "" || amountCents == 0 {
		return 0, errors.New("idempotency key, description, and non-zero amount are required")
	}
	switch entryType {
	case "credit", "refund":
		if amountCents < 0 {
			return 0, errors.New("credits and refunds must be positive amounts")
		}
	case "debit":
		if amountCents > 0 {
			return 0, errors.New("debits must be negative amounts")
		}
	case "adjustment":
	default:
		return 0, errors.New("unsupported wallet entry type")
	}
	var tripUUID *uuid.UUID
	if tripID != "" {
		parsed, err := uuid.Parse(tripID)
		if err != nil {
			return 0, fmt.Errorf("invalid trip id: %w", err)
		}
		tripUUID = &parsed
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin wallet operation: %w", err)
	}
	defer tx.Rollback(ctx)
	var accountID uuid.UUID
	if err := tx.QueryRow(ctx, queries.Must("lock_wallet_account.sql"), userUUID).Scan(&accountID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("lock wallet: %w", err)
	}
	var oldAmount int64
	var oldType string
	var oldTrip *uuid.UUID
	var oldUser uuid.UUID
	err = tx.QueryRow(ctx, queries.Must("find_wallet_entry_by_key.sql"), key).Scan(&oldUser, &oldAmount, &oldType, &oldTrip)
	if err == nil {
		if oldUser != userUUID || oldAmount != amountCents || oldType != entryType || !sameUUID(oldTrip, tripUUID) {
			return 0, ErrIdempotencyConflict
		}
		var balance int64
		if err := tx.QueryRow(ctx, queries.Must("read_wallet_balance.sql"), userUUID).Scan(&balance); err != nil {
			return 0, fmt.Errorf("read wallet balance: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit idempotent wallet result: %w", err)
		}
		return balance, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("check wallet idempotency: %w", err)
	}
	var balance int64
	if err := tx.QueryRow(ctx, queries.Must("read_wallet_balance.sql"), userUUID).Scan(&balance); err != nil {
		return 0, fmt.Errorf("read wallet balance: %w", err)
	}
	maxInt64 := int64(^uint64(0) >> 1)
	minInt64 := -maxInt64 - 1
	if amountCents > 0 && balance > maxInt64-amountCents || amountCents < 0 && balance < minInt64-amountCents {
		return 0, errors.New("wallet balance exceeds integer range")
	}
	if balance+amountCents < 0 {
		return 0, ErrInsufficientFunds
	}
	if _, err := tx.Exec(ctx, queries.Must("insert_wallet_entry.sql"), userUUID, tripUUID, key, amountCents, entryType, description); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrIdempotencyConflict
		}
		return 0, fmt.Errorf("append wallet entry: %w", err)
	}
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: actorUUID.String(), Action: "wallet.entry_created",
		Resource: "wallet", ResourceID: userUUID.String(), OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"entryType": entryType, "amountCents": fmt.Sprint(amountCents), "idempotencyKey": key},
	}); err != nil {
		return 0, fmt.Errorf("audit wallet mutation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit wallet operation: %w", err)
	}
	return balance + amountCents, nil
}

func (s *PostgresStore) WalletEntries(ctx context.Context, userID string, limit int) ([]models.WalletEntry, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.Pool.Query(ctx, queries.Must("list_wallet_entries.sql"), id, limit)
	if err != nil {
		return nil, fmt.Errorf("list wallet entries: %w", err)
	}
	defer rows.Close()
	entries := make([]models.WalletEntry, 0)
	for rows.Next() {
		var entryID, entryUserID uuid.UUID
		var entry models.WalletEntry
		if err := rows.Scan(&entryID, &entryUserID, &entry.TripID, &entry.IdempotencyKey,
			&entry.AmountCents, &entry.EntryType, &entry.Description, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan wallet entry: %w", err)
		}
		entry.ID = entryID.String()
		entry.UserID = entryUserID.String()
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func sameUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (s *PostgresStore) TripCounts(ctx context.Context) (map[string]interface{}, error) {
	var total, requested, completed, pooled int64
	err := s.Pool.QueryRow(ctx, queries.Must("trip_counts.sql")).Scan(&total, &requested, &completed, &pooled)
	if err != nil {
		return nil, fmt.Errorf("count trips: %w", err)
	}
	poolRate := float64(0)
	if total > 0 {
		poolRate = float64(pooled) / float64(total)
	}
	return map[string]interface{}{
		"tripsTotal":     total,
		"tripsRequested": requested,
		"tripsCompleted": completed,
		"poolTripRate":   poolRate,
	}, nil
}

func (s *PostgresStore) TransitionTrip(ctx context.Context, tripID, userID, role string, requested tripdomain.Status) (models.Trip, error) {
	if s == nil || s.Pool == nil {
		return models.Trip{}, errors.New("postgres is not configured")
	}
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return models.Trip{}, ErrNotFound
	}
	actorUUID, err := uuid.Parse(userID)
	if err != nil {
		return models.Trip{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.Trip{}, fmt.Errorf("begin trip transition: %w", err)
	}
	defer tx.Rollback(ctx)
	var riderID uuid.UUID
	var driverID *uuid.UUID
	var current tripdomain.Status
	var poolID *uuid.UUID
	err = tx.QueryRow(ctx, queries.Must("lock_trip_transition.sql"), tripUUID).Scan(&riderID, &driverID, &current, &poolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Trip{}, ErrNotFound
	}
	if err != nil {
		return models.Trip{}, fmt.Errorf("load trip transition state: %w", err)
	}
	actor := tripdomain.Actor("")
	switch role {
	case "rider":
		if actorUUID != riderID {
			return models.Trip{}, ErrNotFound
		}
		actor = tripdomain.Rider
	case "driver":
		if driverID == nil || actorUUID != *driverID {
			return models.Trip{}, ErrNotFound
		}
		actor = tripdomain.Driver
	case "admin":
		actor = tripdomain.Operator
	default:
		return models.Trip{}, errors.New("role cannot transition trips")
	}
	if _, err := tripdomain.Transition(current, requested, actor); err != nil {
		return models.Trip{}, err
	}
	if _, err := tx.Exec(ctx, queries.Must("update_trip_status.sql"), tripUUID, requested); err != nil {
		return models.Trip{}, fmt.Errorf("update trip status: %w", err)
	}
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: actorUUID.String(), Action: "trip.status_changed",
		Resource: "trip", ResourceID: tripUUID.String(), OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"from": string(current), "to": string(requested)},
	}); err != nil {
		return models.Trip{}, fmt.Errorf("audit trip transition: %w", err)
	}
	if (requested == tripdomain.Completed || requested == tripdomain.Cancelled) && driverID != nil {
		if poolID == nil {
			if _, err := tx.Exec(ctx, queries.Must("release_driver.sql"), *driverID); err != nil {
				return models.Trip{}, fmt.Errorf("release driver: %w", err)
			}
		} else {
			var active int
			if err := tx.QueryRow(ctx, queries.Must("count_active_pool_trips.sql"), *poolID).Scan(&active); err != nil {
				return models.Trip{}, fmt.Errorf("check pool completion: %w", err)
			}
			if active == 0 {
				if _, err := tx.Exec(ctx, queries.Must("release_driver.sql"), *driverID); err != nil {
					return models.Trip{}, fmt.Errorf("release pooled driver: %w", err)
				}
				if _, err := tx.Exec(ctx, queries.Must("complete_trip_pool.sql"), *poolID); err != nil {
					return models.Trip{}, fmt.Errorf("complete pool: %w", err)
				}
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Trip{}, fmt.Errorf("commit trip transition: %w", err)
	}
	return s.TripByID(ctx, tripUUID.String())
}
