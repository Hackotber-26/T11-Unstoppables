package storage

import (
	"context"
	"fmt"

	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/routing"
	"t11-unstoppables/queries"

	"github.com/google/uuid"
)

func (s *PostgresStore) PoolRoutePoints(ctx context.Context, tripID, userID, role string) (routing.Coordinate, []models.PoolRouteStop, error) {
	tripUUID, err := uuid.Parse(tripID)
	if err != nil {
		return routing.Coordinate{}, nil, ErrNotFound
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return routing.Coordinate{}, nil, ErrNotFound
	}
	rows, err := s.Pool.Query(ctx, queries.Must("pool_route_points.sql"), tripUUID, userUUID, role)
	if err != nil {
		return routing.Coordinate{}, nil, fmt.Errorf("load pooled route stops: %w", err)
	}
	defer rows.Close()

	var pickup routing.Coordinate
	stops := make([]models.PoolRouteStop, 0, 4)
	for rows.Next() {
		var stop models.PoolRouteStop
		if err := rows.Scan(&pickup.Latitude, &pickup.Longitude, &stop.Sequence, &stop.Latitude, &stop.Longitude, &stop.IsYourStop); err != nil {
			return routing.Coordinate{}, nil, fmt.Errorf("scan pooled route stop: %w", err)
		}
		stops = append(stops, stop)
	}
	if err := rows.Err(); err != nil {
		return routing.Coordinate{}, nil, fmt.Errorf("read pooled route stops: %w", err)
	}
	if len(stops) == 0 {
		return routing.Coordinate{}, nil, ErrNotFound
	}
	return pickup, stops, nil
}
