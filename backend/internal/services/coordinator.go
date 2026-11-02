package services

import (
	"context"
	"errors"
	"strings"
	"t11-unstoppables/internal/config"
	"t11-unstoppables/internal/fares"
	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/routing"
	"t11-unstoppables/internal/simulation"
	"t11-unstoppables/internal/storage"
	"t11-unstoppables/internal/transit"
	tripdomain "t11-unstoppables/internal/trips"
	"time"
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

func (c *Coordinator) ListVehicles(ctx context.Context) ([]models.Vehicle, error) {
	return c.Store.Postgres.ListVehicles(ctx)
}

func (c *Coordinator) DriverByUserID(ctx context.Context, userID string) (models.Driver, error) {
	return c.Store.Postgres.DriverByUserID(ctx, userID)
}

func (c *Coordinator) UpdateDriverStatus(ctx context.Context, userID, status string) (models.Driver, error) {
	if status != "available" && status != "offline" {
		return models.Driver{}, errors.New("driver availability must be available or offline")
	}
	return c.Store.Postgres.UpdateDriverStatus(ctx, userID, status)
}

func (c *Coordinator) UpdateDriverVerification(ctx context.Context, actorID, userID string, verified bool) (models.Driver, error) {
	return c.Store.Postgres.UpdateDriverVerification(ctx, actorID, userID, verified)
}

func (c *Coordinator) AssignDriverVehicle(ctx context.Context, actorID, userID, vehicleID string) (models.Driver, error) {
	return c.Store.Postgres.AssignDriverVehicle(ctx, actorID, userID, vehicleID)
}

func (c *Coordinator) ListStations(ctx context.Context) ([]models.Station, error) {
	return c.Store.Postgres.ListStations(ctx)
}

func (c *Coordinator) TransitRoute(ctx context.Context, originID, destinationID string, departure time.Time) (transit.Route, error) {
	if c == nil || c.Cfg == nil || c.Store == nil || c.Store.Postgres == nil {
		return transit.Route{}, transit.ErrNotConfigured
	}
	stations, err := c.Store.Postgres.ListStations(ctx)
	if err != nil {
		return transit.Route{}, err
	}
	stationNames := make(map[string]string, len(stations))
	for _, station := range stations {
		stationNames[station.ID] = station.Name
	}
	origin, originFound := stationNames[originID]
	destination, destinationFound := stationNames[destinationID]
	if !originFound || !destinationFound || originID == destinationID {
		return transit.Route{}, storage.ErrInvalidStation
	}
	planner := transit.NewGoogleRoutes(c.Cfg.GoogleMapsAPIKey)
	return planner.Plan(
		ctx,
		origin+", Bengaluru, Karnataka, India",
		destination+", Bengaluru, Karnataka, India",
		departure,
	)
}

func (c *Coordinator) CreateTrip(ctx context.Context, riderID, originID, destination, poolMode string, pickupLat, pickupLng, dropoffLat, dropoffLng float64, partySize int, requiresAccessible bool) (models.Trip, error) {
	route, err := c.NavigationRoute(ctx, pickupLat, pickupLng, dropoffLat, dropoffLng)
	if err != nil {
		return models.Trip{}, err
	}
	quote, err := fares.QuoteFare(route.DistanceMeters, poolMode)
	if err != nil {
		return models.Trip{}, err
	}
	return c.Store.Postgres.CreateTripWithSubscriptionDiscount(ctx, riderID, originID, destination, poolMode, route.DistanceMeters, pickupLat, pickupLng, dropoffLat, dropoffLng, partySize, requiresAccessible, quote.TotalCents)
}

func (c *Coordinator) TripByID(ctx context.Context, id, userID, role string) (models.Trip, error) {
	trip, err := c.Store.Postgres.TripByID(ctx, id)
	if err != nil {
		return models.Trip{}, err
	}
	allowed := role == "admin" || role == "rider" && trip.RiderID == userID || role == "driver" && trip.DriverID == userID
	if !allowed {
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

func (c *Coordinator) FareQuote(ctx context.Context, riderID, tripID string, pickupLat, pickupLng, dropoffLat, dropoffLng float64, poolMode string) (models.FareQuote, error) {
	route, err := c.NavigationRoute(ctx, pickupLat, pickupLng, dropoffLat, dropoffLng)
	if err != nil {
		return models.FareQuote{}, err
	}
	quote, err := fares.QuoteFare(route.DistanceMeters, poolMode)
	if err != nil {
		return models.FareQuote{}, err
	}
	total, subscriptionDiscount, err := c.Store.Postgres.ApplySubscriptionDiscount(ctx, riderID, quote.TotalCents)
	if err != nil {
		return models.FareQuote{}, err
	}
	return models.FareQuote{
		TripID: tripID, BaseFare: float64(quote.BaseCents) / 100,
		PoolDiscount: float64(quote.DiscountCents) / 100, Total: float64(total) / 100,
		BaseCents: quote.BaseCents, DistanceCents: quote.DistanceCents, DiscountCents: quote.DiscountCents,
		SubscriptionDiscountPaise: subscriptionDiscount, TotalCents: total, Currency: "INR",
		DistanceMeters: route.DistanceMeters, DurationSeconds: route.DurationSeconds, Geometry: route.Geometry,
	}, nil
}

func (c *Coordinator) RunSimulation(scenario simulation.Scenario) (simulation.Result, error) {
	return simulation.RunScenario(scenario)
}

func (c *Coordinator) DispatchPending(ctx context.Context) (map[string]int, error) {
	pools, err := c.Store.Postgres.FormPools(ctx)
	if err != nil {
		return nil, err
	}
	soloTrips, err := c.Store.Postgres.DispatchSoloTrips(ctx, 100)
	if err != nil {
		return nil, err
	}
	return map[string]int{"poolsFormed": pools, "soloTripsDispatched": soloTrips}, nil
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
	return map[string]interface{}{"balancePaise": balance, "currency": "INR", "entries": entries}, nil
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
	return map[string]interface{}{"tripId": trip.ID, "status": "paid-from-wallet", "amountPaise": trip.FareCents, "balancePaise": balance}, nil
}

func (c *Coordinator) CreditWallet(ctx context.Context, adminID, userID, key, description string, amountCents int64) (map[string]interface{}, error) {
	balance, err := c.Store.Postgres.AppendWalletEntry(ctx, adminID, userID, "", key, "credit", description, amountCents)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"userId": userID, "creditedBy": adminID, "amountPaise": amountCents, "balancePaise": balance}, nil
}

func (c *Coordinator) CreateSupportTicket(ctx context.Context, userID, category, subject, message string) (models.SupportTicket, error) {
	category = strings.TrimSpace(category)
	subject = strings.TrimSpace(subject)
	message = strings.TrimSpace(message)
	if category != "trip" && category != "wallet" && category != "safety" && category != "other" ||
		subject == "" || len(subject) > 120 || message == "" || len(message) > 2000 {
		return models.SupportTicket{}, errors.New("category, subject, or message is invalid")
	}
	return c.Store.Postgres.CreateSupportTicket(ctx, userID, category, subject, message)
}

func (c *Coordinator) SupportTickets(ctx context.Context, userID string, admin bool) ([]models.SupportTicket, error) {
	return c.Store.Postgres.ListSupportTickets(ctx, userID, admin)
}

func (c *Coordinator) UpdateSupportTicketStatus(ctx context.Context, adminID, ticketID, status string) (models.SupportTicket, error) {
	if status != "open" && status != "in_progress" && status != "resolved" {
		return models.SupportTicket{}, errors.New("support status must be open, in_progress, or resolved")
	}
	return c.Store.Postgres.UpdateSupportTicketStatus(ctx, adminID, ticketID, status)
}

func (c *Coordinator) CreateTripRating(ctx context.Context, riderID, tripID string, stars int, comment string) (models.TripRating, error) {
	comment = strings.TrimSpace(comment)
	if stars < 1 || stars > 5 || len(comment) > 1000 {
		return models.TripRating{}, errors.New("rating must be 1 to 5 stars with a comment up to 1000 characters")
	}
	return c.Store.Postgres.CreateTripRating(ctx, riderID, tripID, stars, comment)
}

func (c *Coordinator) TripRating(ctx context.Context, riderID, tripID string) (models.TripRating, error) {
	return c.Store.Postgres.TripRating(ctx, riderID, tripID)
}

func (c *Coordinator) VerifyUserPhone(ctx context.Context, userID, phone string) error {
	return c.Store.Postgres.VerifyUserPhone(ctx, userID, phone)
}

func (c *Coordinator) SaveDriverLocation(ctx context.Context, driverID string, location models.DriverLocation) error {
	return c.Store.Postgres.SaveDriverLocation(ctx, driverID, location)
}

func (c *Coordinator) TripLocation(ctx context.Context, tripID, userID, role string) (models.DriverLocation, error) {
	return c.Store.Postgres.TripLocation(ctx, tripID, userID, role)
}

func (c *Coordinator) NavigationRoute(ctx context.Context, startLat, startLng, endLat, endLng float64) (routing.NavigationRoute, error) {
	if c.Cfg == nil || c.Cfg.OSRMBaseURL == "" {
		return routing.NavigationRoute{}, routing.ErrRouterUnavailable
	}
	return routing.OSRMRoute(ctx, c.Cfg.OSRMBaseURL, startLat, startLng, endLat, endLng)
}

func (c *Coordinator) PoolRoutePreview(ctx context.Context, tripID, userID, role string) (models.PoolRoutePreview, error) {
	if c.Cfg == nil || c.Cfg.OSRMBaseURL == "" {
		return models.PoolRoutePreview{}, routing.ErrRouterUnavailable
	}
	pickup, stops, err := c.Store.Postgres.PoolRoutePoints(ctx, tripID, userID, role)
	if err != nil {
		return models.PoolRoutePreview{}, err
	}
	yourStop := -1
	points := []routing.Coordinate{pickup}
	result := models.PoolRoutePreview{Stops: stops}
	for i, stop := range stops {
		points = append(points, routing.Coordinate{Latitude: stop.Latitude, Longitude: stop.Longitude})
		if stop.IsYourStop {
			yourStop = i
		}
	}
	if yourStop < 0 {
		return models.PoolRoutePreview{}, storage.ErrNotFound
	}
	direct, err := routing.OSRMRoute(ctx, c.Cfg.OSRMBaseURL, pickup.Latitude, pickup.Longitude, stops[yourStop].Latitude, stops[yourStop].Longitude)
	if err != nil {
		return models.PoolRoutePreview{}, err
	}
	shared, err := routing.OSRMRouteThrough(ctx, c.Cfg.OSRMBaseURL, points)
	if err != nil {
		return models.PoolRoutePreview{}, err
	}
	throughYourStop, err := routing.OSRMRouteThrough(ctx, c.Cfg.OSRMBaseURL, points[:yourStop+2])
	if err != nil {
		return models.PoolRoutePreview{}, err
	}
	result.DirectDistanceMeters = direct.DistanceMeters
	result.DirectDurationSeconds = direct.DurationSeconds
	result.SharedDistanceMeters = throughYourStop.DistanceMeters
	result.SharedDurationSeconds = throughYourStop.DurationSeconds
	result.PoolRouteDistanceMeters = shared.DistanceMeters
	result.PoolRouteDurationSeconds = shared.DurationSeconds
	result.DistanceDeviationMeters = max(0, throughYourStop.DistanceMeters-direct.DistanceMeters)
	result.TimeWastedSeconds = max(0, throughYourStop.DurationSeconds-direct.DurationSeconds)
	result.Geometry = shared.Geometry
	return result, nil
}

func (c *Coordinator) SaveDriverDocument(ctx context.Context, driverID, documentType, filename, contentType string, data []byte) (models.DriverDocument, error) {
	return c.Store.Postgres.SaveDriverDocument(ctx, driverID, documentType, filename, contentType, c.Cfg.DocumentsPath, data, c.Cfg.DocumentsEncryptionKey)
}

func (c *Coordinator) DriverDocuments(ctx context.Context, driverID string, all bool) ([]models.DriverDocument, error) {
	return c.Store.Postgres.ListDriverDocuments(ctx, driverID, all)
}

func (c *Coordinator) ReviewDriverDocument(ctx context.Context, adminID, documentID, status, note string) (models.DriverDocument, error) {
	return c.Store.Postgres.ReviewDriverDocument(ctx, adminID, documentID, status, note)
}

func (c *Coordinator) ReadDriverDocument(ctx context.Context, documentID string) (models.DriverDocument, []byte, error) {
	return c.Store.Postgres.ReadDriverDocument(ctx, documentID, c.Cfg.DocumentsPath, c.Cfg.DocumentsEncryptionKey)
}

func (c *Coordinator) DriverEarnings(ctx context.Context, driverID string) (models.DriverEarnings, error) {
	return c.Store.Postgres.DriverEarnings(ctx, driverID)
}

func (c *Coordinator) DriverRatingSummary(ctx context.Context, driverID string) (models.DriverRatingSummary, error) {
	return c.Store.Postgres.DriverRatingSummary(ctx, driverID)
}

func (c *Coordinator) ListBays(ctx context.Context) ([]models.StationBay, error) {
	return c.Store.Postgres.ListBays(ctx)
}

func (c *Coordinator) SaveBay(ctx context.Context, bay models.StationBay) (models.StationBay, error) {
	return c.Store.Postgres.SaveBay(ctx, bay)
}

func (c *Coordinator) AssignBay(ctx context.Context, driverID, tripID string) (models.BayAssignment, error) {
	return c.Store.Postgres.AssignBay(ctx, driverID, tripID)
}

func (c *Coordinator) MarkBayArrival(ctx context.Context, driverID, tripID string) (models.BayAssignment, error) {
	return c.Store.Postgres.MarkBayArrival(ctx, driverID, tripID)
}

func (c *Coordinator) SubscriptionPlans(ctx context.Context, includeInactive bool) ([]models.SubscriptionPlan, error) {
	return c.Store.Postgres.ListSubscriptionPlans(ctx, includeInactive)
}

func (c *Coordinator) SaveSubscriptionPlan(ctx context.Context, plan models.SubscriptionPlan) (models.SubscriptionPlan, error) {
	return c.Store.Postgres.SaveSubscriptionPlan(ctx, plan)
}

func (c *Coordinator) SetSubscriptionPlanActive(ctx context.Context, planID string, active bool) error {
	return c.Store.Postgres.SetSubscriptionPlanActive(ctx, planID, active)
}

func (c *Coordinator) Subscribe(ctx context.Context, userID, planID, idempotencyKey string) (models.UserSubscription, error) {
	return c.Store.Postgres.Subscribe(ctx, userID, planID, idempotencyKey)
}

func (c *Coordinator) CurrentSubscription(ctx context.Context, userID string) (models.UserSubscription, error) {
	return c.Store.Postgres.CurrentSubscription(ctx, userID)
}

func (c *Coordinator) CancelSubscription(ctx context.Context, userID, subscriptionID string) error {
	return c.Store.Postgres.CancelSubscription(ctx, userID, subscriptionID)
}

func (c *Coordinator) SubscriptionDiscount(ctx context.Context, userID string, farePaise int64) (int64, int64, error) {
	return c.Store.Postgres.ApplySubscriptionDiscount(ctx, userID, farePaise)
}
