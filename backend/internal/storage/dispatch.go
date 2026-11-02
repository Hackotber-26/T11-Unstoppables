package storage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"t11-unstoppables/internal/fares"
	"t11-unstoppables/internal/safher"
	"t11-unstoppables/queries"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxPoolMembers = 4

type pendingTrip struct {
	ID          uuid.UUID
	RiderID     uuid.UUID
	OriginID    string
	Destination string
	CreatedAt   time.Time
	PickupLat   float64
	PickupLng   float64
	DropoffLat  float64
	DropoffLng  float64
	PartySize   int
	Distance    float64
	PoolMode    string
}

func (s *PostgresStore) FormPools(ctx context.Context) (int, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin pool formation: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, queries.Must("pending_pool_requests.sql"))
	if err != nil {
		return 0, fmt.Errorf("load pending pool requests: %w", err)
	}
	requests := make([]pendingTrip, 0)
	for rows.Next() {
		var trip pendingTrip
		if err := rows.Scan(&trip.ID, &trip.RiderID, &trip.OriginID, &trip.Destination, &trip.CreatedAt,
			&trip.PickupLat, &trip.PickupLng, &trip.DropoffLat, &trip.DropoffLng, &trip.PartySize); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan pending pool request: %w", err)
		}
		requests = append(requests, trip)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	groups := poolableTripGroups(requests)
	formed := 0
	for _, members := range groups {
		if len(members) >= 2 {
			var driverID uuid.UUID
			var driverVerified, vehicleVerified bool
			requiredSeats := 0
			for _, member := range members {
				requiredSeats += member.PartySize
			}
			err := tx.QueryRow(ctx, queries.Must("select_pool_driver.sql"), requiredSeats).Scan(&driverID, &driverVerified, &vehicleVerified)
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			if err != nil {
				return 0, fmt.Errorf("select pool driver: %w", err)
			}
			decision, err := safher.Evaluate(safher.Context{
				DriverVerified: driverVerified, VehicleVerified: vehicleVerified,
				MaxRouteRisk: 1, RouteRisk: 0.1, MaxWalkingMeters: 500,
			})
			if err != nil || decision.Outcome != safher.Allow {
				return 0, fmt.Errorf("pool failed safety policy: %v", err)
			}
			var poolID uuid.UUID
			if err := tx.QueryRow(ctx, queries.Must("create_trip_pool.sql"), members[0].OriginID, members[0].Destination, driverID).Scan(&poolID); err != nil {
				return 0, fmt.Errorf("create trip pool: %w", err)
			}
			for index, member := range members {
				if _, err := tx.Exec(ctx, queries.Must("assign_pooled_trip.sql"), member.ID, driverID, poolID); err != nil {
					return 0, fmt.Errorf("assign pooled trip: %w", err)
				}
				if _, err := tx.Exec(ctx, queries.Must("insert_trip_pool_member.sql"), poolID, member.ID, index+1); err != nil {
					return 0, fmt.Errorf("add trip pool member: %w", err)
				}
			}
			if _, err := tx.Exec(ctx, queries.Must("mark_driver_busy.sql"), driverID); err != nil {
				return 0, fmt.Errorf("mark pool driver busy: %w", err)
			}
			formed++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit pool formation: %w", err)
	}
	return formed, nil
}

func poolableTripGroups(requests []pendingTrip) [][]pendingTrip {
	sort.SliceStable(requests, func(i, j int) bool {
		if requests[i].OriginID != requests[j].OriginID {
			return requests[i].OriginID < requests[j].OriginID
		}
		if !requests[i].CreatedAt.Equal(requests[j].CreatedAt) {
			return requests[i].CreatedAt.Before(requests[j].CreatedAt)
		}
		return requests[i].ID.String() < requests[j].ID.String()
	})
	groups := make([][]pendingTrip, 0)
	for _, trip := range requests {
		placed := false
		for i := range groups {
			group := groups[i]
			first := group[0]
			seats := trip.PartySize
			for _, member := range group {
				seats += member.PartySize
			}
			if first.OriginID != trip.OriginID || len(group) >= maxPoolMembers || seats > 8 ||
				distanceBetween(trip.PickupLat, trip.PickupLng, first.PickupLat, first.PickupLng) > 500 ||
				distanceBetween(trip.DropoffLat, trip.DropoffLng, first.DropoffLat, first.DropoffLng) > 5000 {
				continue
			}
			duplicateRider := false
			for _, member := range group {
				if member.RiderID == trip.RiderID {
					duplicateRider = true
					break
				}
			}
			if duplicateRider {
				continue
			}
			groups[i] = append(group, trip)
			placed = true
			break
		}
		if !placed {
			groups = append(groups, []pendingTrip{trip})
		}
	}
	return groups
}

func distanceBetween(latitudeA, longitudeA, latitudeB, longitudeB float64) float64 {
	const earthRadiusMeters = 6371000
	toRadians := func(value float64) float64 { return value * math.Pi / 180 }
	latA, latB := toRadians(latitudeA), toRadians(latitudeB)
	deltaLat, deltaLng := latB-latA, toRadians(longitudeB-longitudeA)
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latA)*math.Cos(latB)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (s *PostgresStore) DispatchSoloTrips(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin trip dispatch: %w", err)
	}
	defer tx.Rollback(ctx)
	dispatched := 0
	for dispatched < limit {
		var trip pendingTrip
		err := tx.QueryRow(ctx, queries.Must("pending_solo_trip.sql")).Scan(
			&trip.ID, &trip.RiderID, &trip.OriginID, &trip.Destination, &trip.CreatedAt, &trip.Distance, &trip.PoolMode,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("select solo trip: %w", err)
		}
		var driverID uuid.UUID
		var driverVerified, vehicleVerified bool
		err = tx.QueryRow(ctx, queries.Must("select_solo_driver.sql")).Scan(&driverID, &driverVerified, &vehicleVerified)
		if errors.Is(err, pgx.ErrNoRows) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("select solo driver: %w", err)
		}
		decision, err := safher.Evaluate(safher.Context{
			DriverVerified: driverVerified, VehicleVerified: vehicleVerified,
			MaxRouteRisk: 1, RouteRisk: trip.Distance / 500000, MaxWalkingMeters: 500,
		})
		if err != nil || decision.Outcome != safher.Allow {
			return 0, fmt.Errorf("solo trip failed safety policy: %v", err)
		}
		poolMode := trip.PoolMode
		fareCents := int64(0)
		if poolMode == "corridor" {
			quote, err := fares.QuoteFare(trip.Distance, "solo")
			if err != nil {
				return 0, fmt.Errorf("reprice expired pool request: %w", err)
			}
			fareCents = quote.TotalCents
			poolMode = "solo"
		}
		reprice := trip.PoolMode == "corridor"
		if _, err := tx.Exec(ctx, queries.Must("dispatch_solo_trip.sql"), trip.ID, driverID, poolMode, reprice, fareCents); err != nil {
			return 0, fmt.Errorf("assign solo trip: %w", err)
		}
		if _, err := tx.Exec(ctx, queries.Must("mark_driver_busy.sql"), driverID); err != nil {
			return 0, fmt.Errorf("mark solo driver busy: %w", err)
		}
		dispatched++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit trip dispatch: %w", err)
	}
	return dispatched, nil
}
