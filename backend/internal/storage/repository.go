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
var ErrDriverBusy = errors.New("driver availability cannot change during an in-progress trip")
var ErrDriverNotVerified = errors.New("driver must be verified before becoming available")
var ErrVehicleUnavailable = errors.New("vehicle must be active, verified, and unassigned")
var ErrInvalidStation = errors.New("origin station is not active in the metro catalog")
var ErrInvalidInvitation = errors.New("account invitation is invalid or expired")
var ErrTripNotCompleted = errors.New("only completed trips can be rated")
var ErrAlreadyRated = errors.New("trip already has a rating")

func (s *PostgresStore) CreateAccount(ctx context.Context, name, email, phone, passwordHash, role, invitationHash string) (models.User, error) {
	if s == nil || s.Pool == nil {
		return models.User{}, errors.New("postgres is not configured")
	}
	if role != "rider" && role != "driver" && role != "admin" {
		return models.User{}, errors.New("unsupported account role")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("begin account registration: %w", err)
	}
	defer tx.Rollback(ctx)

	var invitedBy *uuid.UUID
	if role != "rider" {
		var issuer uuid.UUID
		err = tx.QueryRow(ctx, `
			UPDATE account_invitations
			SET redeemed_at = NOW()
			WHERE token_hash = $1 AND role = $2 AND redeemed_at IS NULL AND expires_at > NOW()
			RETURNING created_by
		`, invitationHash, role).Scan(&issuer)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrInvalidInvitation
		}
		if err != nil {
			return models.User{}, fmt.Errorf("redeem account invitation: %w", err)
		}
		invitedBy = &issuer
	}

	var id uuid.UUID
	var user models.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, phone)
		VALUES ($1, LOWER($2), $3, $4, $5)
		RETURNING id, name, email, role, created_at, phone, phone_verified
	`,
		strings.TrimSpace(name), strings.TrimSpace(email), passwordHash, role, strings.TrimSpace(phone),
	).Scan(&id, &user.Name, &user.Email, &user.Role, &user.CreatedAt, &user.Phone, &user.PhoneVerified)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, ErrConflict
		}
		return models.User{}, fmt.Errorf("insert account: %w", err)
	}
	user.ID = id.String()
	switch role {
	case "rider":
		if _, err := tx.Exec(ctx, queries.Must("create_rider_profile.sql"), id, strings.TrimSpace(phone)); err != nil {
			return models.User{}, fmt.Errorf("insert rider profile: %w", err)
		}
		if _, err := tx.Exec(ctx, queries.Must("create_wallet_account.sql"), id); err != nil {
			return models.User{}, fmt.Errorf("create rider wallet: %w", err)
		}
	case "driver":
		if _, err := tx.Exec(ctx, "INSERT INTO driver_profiles (user_id) VALUES ($1)", id); err != nil {
			return models.User{}, fmt.Errorf("insert driver profile: %w", err)
		}
	}
	actorID := id.String()
	if invitedBy != nil {
		actorID = invitedBy.String()
	}
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: actorID, Action: "user.registered",
		Resource: "user", ResourceID: id.String(), OccurredAt: time.Now().UTC(),
	}); err != nil {
		return models.User{}, fmt.Errorf("audit account registration: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.User{}, fmt.Errorf("commit account registration: %w", err)
	}
	return user, nil
}

func (s *PostgresStore) CreateRider(ctx context.Context, name, email, phone, passwordHash string) (models.User, error) {
	return s.CreateAccount(ctx, name, email, phone, passwordHash, "rider", "")
}

func (s *PostgresStore) CreateInvitation(ctx context.Context, role, createdBy, tokenHash string, expiresAt time.Time) error {
	if s == nil || s.Pool == nil {
		return errors.New("postgres is not configured")
	}
	if role != "driver" && role != "admin" {
		return errors.New("invitations must grant driver or admin access")
	}
	if len(tokenHash) != 64 || !expiresAt.After(time.Now()) {
		return errors.New("invitation hash and future expiry are required")
	}
	issuer, err := uuid.Parse(createdBy)
	if err != nil {
		return fmt.Errorf("invalid invitation issuer: %w", err)
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin account invitation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_invitations (token_hash, role, created_by, expires_at)
		VALUES ($1, $2, $3, $4)
	`, tokenHash, role, issuer, expiresAt); err != nil {
		return fmt.Errorf("create account invitation: %w", err)
	}
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: issuer.String(), Action: "account.invitation_created",
		Resource: "account_invitation", ResourceID: tokenHash[:12], OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"role": role, "expiresAt": expiresAt.UTC().Format(time.RFC3339)},
	}); err != nil {
		return fmt.Errorf("audit account invitation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account invitation: %w", err)
	}
	return nil
}

func (s *PostgresStore) UserByEmail(ctx context.Context, email string) (models.User, error) {
	if s == nil || s.Pool == nil {
		return models.User{}, errors.New("postgres is not configured")
	}
	var id uuid.UUID
	var user models.User
	err := s.Pool.QueryRow(ctx, queries.Must("find_user_by_email.sql"), strings.TrimSpace(email)).Scan(
		&id, &user.Name, &user.Email, &user.Role, &user.PasswordHash, &user.CreatedAt, &user.Phone, &user.PhoneVerified,
	)
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
		if err := rows.Scan(&id, &driver.Name, &driver.VehicleID, &driver.Status, &driver.Capacity, &driver.Verified); err != nil {
			return nil, fmt.Errorf("scan driver: %w", err)
		}
		driver.ID = id.String()
		result = append(result, driver)
	}
	return result, rows.Err()
}

func (s *PostgresStore) ListVehicles(ctx context.Context) ([]models.Vehicle, error) {
	if s == nil || s.Pool == nil {
		return nil, errors.New("postgres is not configured")
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT v.id, v.label, v.vehicle_type, v.capacity, v.active, v.verified,
		       v.accessibility_supported,
		       COALESCE((SELECT d.status FROM driver_profiles d WHERE d.vehicle_id = v.id LIMIT 1), 'unassigned')
		FROM fleet_vehicles v
		ORDER BY v.created_at DESC, v.label
	`)
	if err != nil {
		return nil, fmt.Errorf("list fleet vehicles: %w", err)
	}
	defer rows.Close()
	vehicles := make([]models.Vehicle, 0)
	for rows.Next() {
		var id uuid.UUID
		var vehicle models.Vehicle
		if err := rows.Scan(&id, &vehicle.Label, &vehicle.Type, &vehicle.Capacity, &vehicle.Active,
			&vehicle.Verified, &vehicle.AccessibilitySupported, &vehicle.DriverStatus); err != nil {
			return nil, fmt.Errorf("scan fleet vehicle: %w", err)
		}
		vehicle.ID = id.String()
		vehicles = append(vehicles, vehicle)
	}
	return vehicles, rows.Err()
}

func (s *PostgresStore) DriverByUserID(ctx context.Context, userID string) (models.Driver, error) {
	if s == nil || s.Pool == nil {
		return models.Driver{}, errors.New("postgres is not configured")
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	var driverID uuid.UUID
	var driver models.Driver
	err = s.Pool.QueryRow(ctx, queries.Must("get_driver_by_user_id.sql"), id).Scan(
		&driverID, &driver.Name, &driver.VehicleID, &driver.Status, &driver.Capacity, &driver.Verified,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Driver{}, ErrNotFound
	}
	if err != nil {
		return models.Driver{}, fmt.Errorf("find driver profile: %w", err)
	}
	driver.ID = driverID.String()
	return driver, nil
}

func (s *PostgresStore) UpdateDriverStatus(ctx context.Context, userID, status string) (models.Driver, error) {
	if s == nil || s.Pool == nil {
		return models.Driver{}, errors.New("postgres is not configured")
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	var driverID uuid.UUID
	var driver models.Driver
	err = s.Pool.QueryRow(ctx, queries.Must("update_driver_status.sql"), id, status).Scan(
		&driverID, &driver.Name, &driver.VehicleID, &driver.Status, &driver.Capacity, &driver.Verified,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		var verified bool
		err = s.Pool.QueryRow(ctx, "SELECT verified FROM driver_profiles WHERE user_id = $1", id).Scan(&verified)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Driver{}, ErrNotFound
		}
		if err != nil {
			return models.Driver{}, fmt.Errorf("check driver verification: %w", err)
		}
		if !verified {
			return models.Driver{}, ErrDriverNotVerified
		}
		return models.Driver{}, ErrDriverBusy
	}
	if err != nil {
		return models.Driver{}, fmt.Errorf("update driver availability: %w", err)
	}
	driver.ID = driverID.String()
	return driver, nil
}

func (s *PostgresStore) UpdateDriverVerification(ctx context.Context, actorID, userID string, verified bool) (models.Driver, error) {
	if s == nil || s.Pool == nil {
		return models.Driver{}, errors.New("postgres is not configured")
	}
	id, err := uuid.Parse(userID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.Driver{}, fmt.Errorf("begin driver verification update: %w", err)
	}
	defer tx.Rollback(ctx)
	var driverID uuid.UUID
	var driver models.Driver
	err = tx.QueryRow(ctx, `
		UPDATE driver_profiles d
		SET verified = $2, status = CASE WHEN $2 THEN d.status ELSE 'offline' END
		WHERE d.user_id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM trip_requests t
		    WHERE t.driver_id = d.user_id AND t.status = 'in_progress'
		  )
		RETURNING d.user_id,
		  (SELECT u.name FROM users u WHERE u.id = d.user_id),
		  COALESCE(d.vehicle_id::text, ''),
		  d.status,
		  COALESCE((SELECT v.capacity FROM fleet_vehicles v WHERE v.id = d.vehicle_id), 0),
		  d.verified
	`, id, verified).Scan(&driverID, &driver.Name, &driver.VehicleID, &driver.Status, &driver.Capacity, &driver.Verified)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM driver_profiles WHERE user_id = $1)", id).Scan(&exists); err != nil {
			return models.Driver{}, fmt.Errorf("check driver profile: %w", err)
		}
		if !exists {
			return models.Driver{}, ErrNotFound
		}
		return models.Driver{}, ErrDriverBusy
	}
	if err != nil {
		return models.Driver{}, fmt.Errorf("update driver verification: %w", err)
	}
	driver.ID = driverID.String()
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: actor.String(), Action: "driver.verification_updated",
		Resource: "driver", ResourceID: driver.ID, OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"verified": fmt.Sprint(verified)},
	}); err != nil {
		return models.Driver{}, fmt.Errorf("audit driver verification: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Driver{}, fmt.Errorf("commit driver verification: %w", err)
	}
	return driver, nil
}

func (s *PostgresStore) AssignDriverVehicle(ctx context.Context, actorID, userID, vehicleID string) (models.Driver, error) {
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	driverUUID, err := uuid.Parse(userID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	vehicleUUID, err := uuid.Parse(vehicleID)
	if err != nil {
		return models.Driver{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.Driver{}, fmt.Errorf("begin driver vehicle assignment: %w", err)
	}
	defer tx.Rollback(ctx)
	var active, vehicleVerified bool
	err = tx.QueryRow(ctx, "SELECT active, verified FROM fleet_vehicles WHERE id = $1 FOR UPDATE", vehicleUUID).Scan(&active, &vehicleVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Driver{}, ErrNotFound
	}
	if err != nil {
		return models.Driver{}, fmt.Errorf("load vehicle for assignment: %w", err)
	}
	if !active || !vehicleVerified {
		return models.Driver{}, ErrVehicleUnavailable
	}
	var assigned bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM driver_profiles WHERE vehicle_id = $1 AND user_id <> $2)", vehicleUUID, driverUUID).Scan(&assigned); err != nil {
		return models.Driver{}, fmt.Errorf("check vehicle assignment: %w", err)
	}
	if assigned {
		return models.Driver{}, ErrVehicleUnavailable
	}
	var driverID uuid.UUID
	var driver models.Driver
	err = tx.QueryRow(ctx, `
		UPDATE driver_profiles d SET vehicle_id = $2
		WHERE d.user_id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM trip_requests t
		    WHERE t.driver_id = d.user_id AND t.status = 'in_progress'
		  )
		RETURNING d.user_id,
		  (SELECT u.name FROM users u WHERE u.id = d.user_id),
		  COALESCE(d.vehicle_id::text, ''),
		  d.status,
		  COALESCE((SELECT v.capacity FROM fleet_vehicles v WHERE v.id = d.vehicle_id), 0),
		  d.verified
	`, driverUUID, vehicleUUID).Scan(&driverID, &driver.Name, &driver.VehicleID, &driver.Status, &driver.Capacity, &driver.Verified)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM driver_profiles WHERE user_id = $1)", driverUUID).Scan(&exists); err != nil {
			return models.Driver{}, fmt.Errorf("check driver profile: %w", err)
		}
		if !exists {
			return models.Driver{}, ErrNotFound
		}
		return models.Driver{}, ErrDriverBusy
	}
	if err != nil {
		return models.Driver{}, fmt.Errorf("assign driver vehicle: %w", err)
	}
	driver.ID = driverID.String()
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: actor.String(), Action: "driver.vehicle_assigned",
		Resource: "driver", ResourceID: driver.ID, OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"vehicleId": vehicleUUID.String()},
	}); err != nil {
		return models.Driver{}, fmt.Errorf("audit driver vehicle assignment: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Driver{}, fmt.Errorf("commit driver vehicle assignment: %w", err)
	}
	return driver, nil
}

func (s *PostgresStore) ListStations(ctx context.Context) ([]models.Station, error) {
	if s == nil || s.Pool == nil {
		return nil, errors.New("postgres is not configured")
	}
	rows, err := s.Pool.Query(ctx, queries.Must("list_stations.sql"))
	if err != nil {
		return nil, fmt.Errorf("list stations: %w", err)
	}
	defer rows.Close()
	result := make([]models.Station, 0)
	for rows.Next() {
		var station models.Station
		if err := rows.Scan(&station.ID, &station.Name, &station.Lines); err != nil {
			return nil, fmt.Errorf("scan station: %w", err)
		}
		result = append(result, station)
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
		&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents,
		&trip.PartySize, &trip.RequiresAccessible, &trip.CreatedAt, &trip.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Trip{}, ErrInvalidStation
		}
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
		&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents,
		&trip.PartySize, &trip.RequiresAccessible, &trip.CreatedAt, &trip.UpdatedAt, &trip.PickupLatitude, &trip.PickupLongitude,
		&trip.DropoffLatitude, &trip.DropoffLongitude)
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
			&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents,
			&trip.PartySize, &trip.RequiresAccessible, &trip.CreatedAt, &trip.UpdatedAt, &trip.PickupLatitude, &trip.PickupLongitude,
			&trip.DropoffLatitude, &trip.DropoffLongitude); err != nil {
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

func (s *PostgresStore) CreateSupportTicket(ctx context.Context, userID, category, subject, message string) (models.SupportTicket, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return models.SupportTicket{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.SupportTicket{}, fmt.Errorf("begin support ticket: %w", err)
	}
	defer tx.Rollback(ctx)
	var ticket models.SupportTicket
	var ticketID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO support_tickets (user_id, category, subject, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, category, subject, message, status, created_at, updated_at
	`, userUUID, category, strings.TrimSpace(subject), strings.TrimSpace(message)).Scan(
		&ticketID, &ticket.UserID, &ticket.Category, &ticket.Subject, &ticket.Message,
		&ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if err != nil {
		return models.SupportTicket{}, fmt.Errorf("insert support ticket: %w", err)
	}
	ticket.ID = ticketID.String()
	ticket.UserID = userUUID.String()
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: userUUID.String(), Action: "support.ticket_created",
		Resource: "support_ticket", ResourceID: ticket.ID, OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"category": category},
	}); err != nil {
		return models.SupportTicket{}, fmt.Errorf("audit support ticket: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.SupportTicket{}, fmt.Errorf("commit support ticket: %w", err)
	}
	return ticket, nil
}

func (s *PostgresStore) ListSupportTickets(ctx context.Context, userID string, admin bool) ([]models.SupportTicket, error) {
	var userFilter *uuid.UUID
	if !admin {
		id, err := uuid.Parse(userID)
		if err != nil {
			return nil, ErrNotFound
		}
		userFilter = &id
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, user_id, category, subject, message, status, created_at, updated_at
		FROM support_tickets
		WHERE ($1::uuid IS NULL OR user_id = $1)
		ORDER BY created_at DESC
		LIMIT 100
	`, userFilter)
	if err != nil {
		return nil, fmt.Errorf("list support tickets: %w", err)
	}
	defer rows.Close()
	tickets := make([]models.SupportTicket, 0)
	for rows.Next() {
		var id, ownerID uuid.UUID
		var ticket models.SupportTicket
		if err := rows.Scan(&id, &ownerID, &ticket.Category, &ticket.Subject, &ticket.Message,
			&ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan support ticket: %w", err)
		}
		ticket.ID = id.String()
		ticket.UserID = ownerID.String()
		tickets = append(tickets, ticket)
	}
	return tickets, rows.Err()
}

func (s *PostgresStore) UpdateSupportTicketStatus(ctx context.Context, actorID, ticketID, status string) (models.SupportTicket, error) {
	actor, err := uuid.Parse(actorID)
	if err != nil {
		return models.SupportTicket{}, ErrNotFound
	}
	id, err := uuid.Parse(ticketID)
	if err != nil {
		return models.SupportTicket{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.SupportTicket{}, fmt.Errorf("begin support status update: %w", err)
	}
	defer tx.Rollback(ctx)
	var ticket models.SupportTicket
	var ticketUUID, ownerID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE support_tickets
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, category, subject, message, status, created_at, updated_at
	`, id, status).Scan(
		&ticketUUID, &ownerID, &ticket.Category, &ticket.Subject, &ticket.Message,
		&ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.SupportTicket{}, ErrNotFound
	}
	if err != nil {
		return models.SupportTicket{}, fmt.Errorf("update support ticket status: %w", err)
	}
	ticket.ID = ticketUUID.String()
	ticket.UserID = ownerID.String()
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: actor.String(), Action: "support.ticket_status_updated",
		Resource: "support_ticket", ResourceID: ticket.ID, OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"status": status},
	}); err != nil {
		return models.SupportTicket{}, fmt.Errorf("audit support ticket update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.SupportTicket{}, fmt.Errorf("commit support ticket update: %w", err)
	}
	return ticket, nil
}

func (s *PostgresStore) CreateTripRating(ctx context.Context, riderID, tripID string, stars int, comment string) (models.TripRating, error) {
	riderUUID, err := uuid.Parse(riderID)
	if err != nil {
		return models.TripRating{}, ErrNotFound
	}
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return models.TripRating{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.TripRating{}, fmt.Errorf("begin trip rating: %w", err)
	}
	defer tx.Rollback(ctx)
	var tripRiderID uuid.UUID
	var driverID *uuid.UUID
	var status string
	err = tx.QueryRow(ctx, "SELECT rider_id, driver_id, status FROM trip_requests WHERE id = $1 FOR UPDATE", tripUUID).
		Scan(&tripRiderID, &driverID, &status)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && tripRiderID != riderUUID {
		return models.TripRating{}, ErrNotFound
	}
	if err != nil {
		return models.TripRating{}, fmt.Errorf("load trip for rating: %w", err)
	}
	if status != "completed" || driverID == nil {
		return models.TripRating{}, ErrTripNotCompleted
	}
	var rating models.TripRating
	var ratingID, ratingTripID, ratingRiderID, ratingDriverID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO trip_ratings (trip_id, rider_id, driver_id, stars, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, trip_id, rider_id, driver_id, stars, comment, created_at
	`, tripUUID, riderUUID, *driverID, stars, strings.TrimSpace(comment)).Scan(
		&ratingID, &ratingTripID, &ratingRiderID, &ratingDriverID,
		&rating.Stars, &rating.Comment, &rating.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.TripRating{}, ErrAlreadyRated
		}
		return models.TripRating{}, fmt.Errorf("insert trip rating: %w", err)
	}
	rating.ID = ratingID.String()
	rating.TripID = ratingTripID.String()
	rating.RiderID = ratingRiderID.String()
	rating.DriverID = ratingDriverID.String()
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: riderUUID.String(), Action: "trip.rated",
		Resource: "trip", ResourceID: tripUUID.String(), OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"stars": fmt.Sprint(stars)},
	}); err != nil {
		return models.TripRating{}, fmt.Errorf("audit trip rating: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.TripRating{}, fmt.Errorf("commit trip rating: %w", err)
	}
	return rating, nil
}

func (s *PostgresStore) TripRating(ctx context.Context, riderID, tripID string) (models.TripRating, error) {
	riderUUID, err := uuid.Parse(riderID)
	if err != nil {
		return models.TripRating{}, ErrNotFound
	}
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return models.TripRating{}, ErrNotFound
	}
	var rating models.TripRating
	var id, ratingTripID, ratingRiderID, ratingDriverID uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		SELECT id, trip_id, rider_id, driver_id, stars, comment, created_at
		FROM trip_ratings
		WHERE rider_id = $1 AND trip_id = $2
	`, riderUUID, tripUUID).Scan(
		&id, &ratingTripID, &ratingRiderID, &ratingDriverID,
		&rating.Stars, &rating.Comment, &rating.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.TripRating{}, ErrNotFound
	}
	if err != nil {
		return models.TripRating{}, fmt.Errorf("load trip rating: %w", err)
	}
	rating.ID = id.String()
	rating.TripID = ratingTripID.String()
	rating.RiderID = ratingRiderID.String()
	rating.DriverID = ratingDriverID.String()
	return rating, nil
}

func sameUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (s *PostgresStore) TripCounts(ctx context.Context) (map[string]interface{}, error) {
	var total, requested, completed, pooled, riders, activeRiders, drivers, activeDrivers, vehicles, activeVehicles, stations, pools int64
	err := s.Pool.QueryRow(ctx, queries.Must("trip_counts.sql")).Scan(
		&total, &requested, &completed, &pooled, &riders, &activeRiders,
		&drivers, &activeDrivers, &vehicles, &activeVehicles, &stations, &pools,
	)
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
		"ridersTotal":    riders,
		"activeRiders":   activeRiders,
		"driversTotal":   drivers,
		"activeDrivers":  activeDrivers,
		"vehiclesTotal":  vehicles,
		"activeVehicles": activeVehicles,
		"stationsTotal":  stations,
		"activePools":    pools,
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
	if requested == tripdomain.Completed || requested == tripdomain.Cancelled {
		if _, err := tx.Exec(ctx, "DELETE FROM driver_locations WHERE trip_id = $1", tripUUID); err != nil {
			return models.Trip{}, fmt.Errorf("clear trip location after completion: %w", err)
		}
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
