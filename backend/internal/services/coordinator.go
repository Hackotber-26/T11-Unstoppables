package services

import (
	"context"
	"errors"
	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/fares"
	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/simulation"
	"t11-unstoppables/internal/storage"
	tripdomain "t11-unstoppables/internal/trips"
)

type Coordinator struct {
	Cfg   *config.Config
	Store *storage.Client
}

func NewCoordinator(cfg *config.Config, store *storage.Client) *Coordinator {
	return &Coordinator{Cfg: cfg, Store: store}
}

func (c *Coordinator) Health(ctx context.Context) models.HealthStatus {
	services := map[string]string{"api": "ok", "database": "unavailable", "redis": "unavailable"}
	status := "degraded"
	if c.Store != nil && c.Store.Postgres != nil && c.Store.Postgres.Pool != nil {
		if err := c.Store.Postgres.Pool.Ping(ctx); err == nil {
			services["database"] = "ready"
			status = "ok"
		}
	}
	if c.Store != nil && c.Store.Redis != nil && c.Store.Redis.Client != nil {
		if err := c.Store.Redis.Client.Ping(ctx).Err(); err == nil {
			services["redis"] = "ready"
		}
	}
	return models.HealthStatus{Status: status, Services: services}
}

func (c *Coordinator) ListRiders(ctx context.Context) ([]models.Rider, error) {
	return c.Store.Postgres.ListRiders(ctx)
}

func (c *Coordinator) ListDrivers(ctx context.Context) ([]models.Driver, error) {
	return c.Store.Postgres.ListDrivers(ctx)
}

func (c *Coordinator) CreateTrip(ctx context.Context, riderID, originID, destination, poolMode string, distanceMeters float64) (models.Trip, error) {
	quote, err := fares.QuoteFare(distanceMeters, poolMode)
	if err != nil {
		return models.Trip{}, err
	}
	return c.Store.Postgres.CreateTrip(ctx, riderID, originID, destination, poolMode, distanceMeters, quote.TotalCents)
}

func (c *Coordinator) TripByID(ctx context.Context, id, userID, role string) (models.Trip, error) {
	trip, err := c.Store.Postgres.TripByID(ctx, id)
	if err != nil {
		return models.Trip{}, err
	}
	if role != "admin" && trip.RiderID != userID {
		return models.Trip{}, storage.ErrNotFound
	}
	return trip, nil
}

func (c *Coordinator) ListTrips(ctx context.Context, userID, role string) ([]models.Trip, error) {
	return c.Store.Postgres.ListTrips(ctx, userID, role, 50)
}

func (c *Coordinator) TransitionTrip(ctx context.Context, tripID, userID, role string, status tripdomain.Status) (models.Trip, error) {
	return c.Store.Postgres.TransitionTrip(ctx, tripID, userID, role, status)
}

func (c *Coordinator) MatchPool(ctx context.Context, riderID string) ([]models.PoolCandidate, error) {
	return c.Store.Postgres.PoolCandidates(ctx, riderID)
}

func (c *Coordinator) FareQuote(tripID string, distanceMeters float64, poolMode string) (models.FareQuote, error) {
	quote, err := fares.QuoteFare(distanceMeters, poolMode)
	if err != nil {
		return models.FareQuote{}, err
	}
	return models.FareQuote{
		TripID: tripID, BaseFare: float64(quote.BaseCents) / 100,
		PoolDiscount: float64(quote.DiscountCents) / 100, Total: float64(quote.TotalCents) / 100,
		BaseCents: quote.BaseCents, DistanceCents: quote.DistanceCents, DiscountCents: quote.DiscountCents,
		TotalCents: quote.TotalCents, Currency: "USD",
	}, nil
}

func (c *Coordinator) RunSimulation(scenario simulation.Scenario) (simulation.Result, error) {
	return simulation.RunScenario(scenario)
}

func (c *Coordinator) Metrics(ctx context.Context) (map[string]interface{}, error) {
	if c.Store == nil || c.Store.Postgres == nil {
		return nil, errors.New("database is unavailable")
	}
	return c.Store.Postgres.TripCounts(ctx)
}

func (c *Coordinator) WalletSummary(ctx context.Context, userID string) (map[string]interface{}, error) {
	balance, err := c.Store.Postgres.WalletBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	entries, err := c.Store.Postgres.WalletEntries(ctx, userID, 50)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"balanceCents": balance, "currency": "USD", "entries": entries}, nil
}

func (c *Coordinator) PayTrip(ctx context.Context, userID, tripID, idempotencyKey string) (map[string]interface{}, error) {
	trip, err := c.Store.Postgres.TripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if trip.RiderID != userID {
		return nil, storage.ErrNotFound
	}
	if trip.Status == "cancelled" {
		return nil, storage.ErrTripNotPayable
	}
	if trip.FareCents <= 0 {
		return nil, errors.New("trip has no payable fare")
	}
	key := idempotencyKey
	if key == "" {
		return nil, errors.New("Idempotency-Key header is required")
	}
	balance, err := c.Store.Postgres.AppendWalletEntry(ctx, userID, userID, trip.ID, key, "debit", "Trip fare payment", -trip.FareCents)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"tripId": trip.ID, "status": "paid-from-wallet", "amountCents": trip.FareCents, "balanceCents": balance}, nil
}

func (c *Coordinator) CreditWallet(ctx context.Context, adminID, userID, key, description string, amountCents int64) (map[string]interface{}, error) {
	balance, err := c.Store.Postgres.AppendWalletEntry(ctx, adminID, userID, "", key, "credit", description, amountCents)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"userId": userID, "creditedBy": adminID, "amountCents": amountCents, "balanceCents": balance}, nil
}
