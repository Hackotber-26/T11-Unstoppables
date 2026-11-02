package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	auditdomain "t11-unstoppables/internal/audit"
	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/subscriptions"
	"t11-unstoppables/queries"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrInvalidSubscriptionPlan = errors.New("subscription plan fields are invalid")
var ErrInvalidSubscriptionRequest = errors.New("subscription request is invalid")
var ErrActiveSubscription = errors.New("rider already has an active subscription")

func (s *PostgresStore) ListSubscriptionPlans(ctx context.Context, includeInactive bool) ([]models.SubscriptionPlan, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, price_paise, monthly_ride_limit, discount_bps, active
		FROM subscription_plans WHERE $1 OR active ORDER BY price_paise, id
	`, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list subscription plans: %w", err)
	}
	defer rows.Close()
	plans := make([]models.SubscriptionPlan, 0)
	for rows.Next() {
		var plan models.SubscriptionPlan
		if err := rows.Scan(&plan.ID, &plan.Name, &plan.PricePaise, &plan.MonthlyRideLimit, &plan.DiscountBPS, &plan.Active); err != nil {
			return nil, fmt.Errorf("scan subscription plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (s *PostgresStore) SaveSubscriptionPlan(ctx context.Context, plan models.SubscriptionPlan) (models.SubscriptionPlan, error) {
	plan.ID = strings.TrimSpace(plan.ID)
	plan.Name = strings.TrimSpace(plan.Name)
	domainPlan := subscriptions.Plan{
		ID: plan.ID, Name: plan.Name, PricePaise: plan.PricePaise,
		MonthlyRideLimit: plan.MonthlyRideLimit, DiscountBPS: int64(plan.DiscountBPS),
	}
	if err := domainPlan.Validate(); err != nil || plan.ID == "" || len(plan.ID) > 40 || plan.Name == "" || len(plan.Name) > 100 {
		return models.SubscriptionPlan{}, ErrInvalidSubscriptionPlan
	}
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO subscription_plans (id, name, price_paise, monthly_ride_limit, discount_bps, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name, price_paise = EXCLUDED.price_paise,
			monthly_ride_limit = EXCLUDED.monthly_ride_limit,
			discount_bps = EXCLUDED.discount_bps, active = EXCLUDED.active
		RETURNING id, name, price_paise, monthly_ride_limit, discount_bps, active
	`, plan.ID, plan.Name, plan.PricePaise, plan.MonthlyRideLimit, plan.DiscountBPS, plan.Active).
		Scan(&plan.ID, &plan.Name, &plan.PricePaise, &plan.MonthlyRideLimit, &plan.DiscountBPS, &plan.Active)
	if err != nil {
		return models.SubscriptionPlan{}, fmt.Errorf("save subscription plan: %w", err)
	}
	return plan, nil
}

func (s *PostgresStore) SetSubscriptionPlanActive(ctx context.Context, planID string, active bool) error {
	result, err := s.Pool.Exec(ctx, "UPDATE subscription_plans SET active = $2 WHERE id = $1", planID, active)
	if err != nil {
		return fmt.Errorf("update subscription plan: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Subscribe(ctx context.Context, userID, planID, idempotencyKey string) (models.UserSubscription, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return models.UserSubscription{}, ErrNotFound
	}
	if idempotencyKey == "" || len(idempotencyKey) > 160 {
		return models.UserSubscription{}, ErrInvalidSubscriptionRequest
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.UserSubscription{}, fmt.Errorf("begin subscription: %w", err)
	}
	defer tx.Rollback(ctx)

	existing, err := subscriptionByKey(ctx, tx, idempotencyKey)
	if err == nil {
		if existing.UserID != userUUID.String() || existing.PlanID != planID {
			return models.UserSubscription{}, ErrIdempotencyConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return models.UserSubscription{}, fmt.Errorf("commit idempotent subscription: %w", err)
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.UserSubscription{}, fmt.Errorf("check subscription idempotency: %w", err)
	}

	var plan models.SubscriptionPlan
	err = tx.QueryRow(ctx, `
		SELECT id, name, price_paise, monthly_ride_limit, discount_bps, active
		FROM subscription_plans WHERE id = $1 FOR SHARE
	`, planID).Scan(&plan.ID, &plan.Name, &plan.PricePaise, &plan.MonthlyRideLimit, &plan.DiscountBPS, &plan.Active)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !plan.Active {
		return models.UserSubscription{}, ErrNotFound
	}
	if err != nil {
		return models.UserSubscription{}, fmt.Errorf("load subscription plan: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE user_subscriptions SET cancelled_at = NOW()
		WHERE user_id = $1 AND cancelled_at IS NULL AND ends_at <= NOW()
	`, userUUID); err != nil {
		return models.UserSubscription{}, fmt.Errorf("expire previous subscriptions: %w", err)
	}
	var walletID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT user_id FROM wallet_accounts WHERE user_id = $1 FOR UPDATE", userUUID).Scan(&walletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserSubscription{}, ErrNotFound
	}
	if err != nil {
		return models.UserSubscription{}, fmt.Errorf("lock subscription wallet: %w", err)
	}
	var existingSubscription uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM user_subscriptions WHERE user_id = $1 AND cancelled_at IS NULL AND ends_at > NOW()
	`, userUUID).Scan(&existingSubscription)
	if err == nil {
		return models.UserSubscription{}, ErrActiveSubscription
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return models.UserSubscription{}, fmt.Errorf("check active subscription: %w", err)
	}
	var balance int64
	if err := tx.QueryRow(ctx, "SELECT COALESCE(SUM(amount_paise), 0) FROM wallet_ledger_entries WHERE user_id = $1", userUUID).Scan(&balance); err != nil {
		return models.UserSubscription{}, fmt.Errorf("read subscription wallet: %w", err)
	}
	if balance < plan.PricePaise {
		return models.UserSubscription{}, ErrInsufficientFunds
	}
	if plan.PricePaise > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO wallet_ledger_entries (user_id, idempotency_key, amount_paise, entry_type, description)
			VALUES ($1, $2, $3, 'debit', $4)
		`, userUUID, idempotencyKey, -plan.PricePaise, "Subscription: "+plan.Name); err != nil {
			return models.UserSubscription{}, fmt.Errorf("charge subscription wallet: %w", err)
		}
	}
	var result models.UserSubscription
	var subID, ownerID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO user_subscriptions (user_id, plan_id, idempotency_key, starts_at, ends_at)
		VALUES ($1, $2, $3, NOW(), NOW() + INTERVAL '30 days')
		RETURNING id, user_id, plan_id, starts_at, ends_at
	`, userUUID, plan.ID, idempotencyKey).Scan(&subID, &ownerID, &result.PlanID, &result.StartsAt, &result.EndsAt)
	if err != nil {
		return models.UserSubscription{}, fmt.Errorf("create subscription: %w", err)
	}
	result.ID, result.UserID, result.PlanName = subID.String(), ownerID.String(), plan.Name
	if err := writeAudit(ctx, tx, auditdomain.Event{
		ID: uuid.NewString(), ActorID: userUUID.String(), Action: "subscription.created",
		Resource: "subscription", ResourceID: result.ID, OccurredAt: time.Now().UTC(),
		Metadata: map[string]string{"planId": plan.ID, "pricePaise": fmt.Sprint(plan.PricePaise)},
	}); err != nil {
		return models.UserSubscription{}, fmt.Errorf("audit subscription: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.UserSubscription{}, fmt.Errorf("commit subscription: %w", err)
	}
	return result, nil
}

func subscriptionByKey(ctx context.Context, tx pgx.Tx, key string) (models.UserSubscription, error) {
	var result models.UserSubscription
	var id, userID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.plan_id, p.name, s.starts_at, s.ends_at, s.cancelled_at IS NOT NULL
		FROM user_subscriptions s JOIN subscription_plans p ON p.id = s.plan_id
		WHERE s.idempotency_key = $1
	`, key).Scan(&id, &userID, &result.PlanID, &result.PlanName, &result.StartsAt, &result.EndsAt, &result.Cancelled)
	if err == nil {
		result.ID, result.UserID = id.String(), userID.String()
	}
	return result, err
}

func (s *PostgresStore) CurrentSubscription(ctx context.Context, userID string) (models.UserSubscription, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return models.UserSubscription{}, ErrNotFound
	}
	var result models.UserSubscription
	var subID, ownerID uuid.UUID
	err = s.Pool.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.plan_id, p.name, s.starts_at, s.ends_at, FALSE
		FROM user_subscriptions s JOIN subscription_plans p ON p.id = s.plan_id
		WHERE s.user_id = $1 AND s.cancelled_at IS NULL AND s.ends_at > NOW()
		ORDER BY s.starts_at DESC LIMIT 1
	`, id).Scan(&subID, &ownerID, &result.PlanID, &result.PlanName, &result.StartsAt, &result.EndsAt, &result.Cancelled)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.UserSubscription{}, ErrNotFound
	}
	if err != nil {
		return models.UserSubscription{}, fmt.Errorf("load current subscription: %w", err)
	}
	result.ID, result.UserID = subID.String(), ownerID.String()
	return result, nil
}

func (s *PostgresStore) CancelSubscription(ctx context.Context, userID, subscriptionID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotFound
	}
	id, err := uuid.Parse(subscriptionID)
	if err != nil {
		return ErrNotFound
	}
	result, err := s.Pool.Exec(ctx, `
		UPDATE user_subscriptions SET cancelled_at = NOW()
		WHERE id = $1 AND user_id = $2 AND cancelled_at IS NULL AND ends_at > NOW()
	`, id, userUUID)
	if err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ApplySubscriptionDiscount(ctx context.Context, userID string, farePaise int64) (int64, int64, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return farePaise, 0, ErrNotFound
	}
	var plan subscriptions.Plan
	var ridesUsed int
	err = s.Pool.QueryRow(ctx, `
		SELECT p.id, p.name, p.price_paise, p.monthly_ride_limit, p.discount_bps,
		       (SELECT COUNT(*) FROM trip_requests t
		        WHERE t.rider_id = s.user_id AND t.status <> 'cancelled'
		          AND t.created_at >= GREATEST(date_trunc('month', NOW()), s.starts_at))
		FROM user_subscriptions s JOIN subscription_plans p ON p.id = s.plan_id
		WHERE s.user_id = $1 AND s.cancelled_at IS NULL AND s.ends_at > NOW()
		ORDER BY s.starts_at DESC LIMIT 1
	`, id).Scan(&plan.ID, &plan.Name, &plan.PricePaise, &plan.MonthlyRideLimit, &plan.DiscountBPS, &ridesUsed)
	if errors.Is(err, pgx.ErrNoRows) {
		return farePaise, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("load active subscription discount: %w", err)
	}
	discounted, err := subscriptions.ApplyDiscount(farePaise, plan, ridesUsed)
	if err != nil {
		return 0, 0, fmt.Errorf("apply subscription discount: %w", err)
	}
	return discounted, farePaise - discounted, nil
}

func (s *PostgresStore) CreateTripWithSubscriptionDiscount(ctx context.Context, riderID, originID, destination, poolMode string, distanceMeters, pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude float64, partySize int, requiresAccessible bool, baseFarePaise int64) (models.Trip, error) {
	riderUUID, err := uuid.Parse(riderID)
	if err != nil {
		return models.Trip{}, ErrNotFound
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.Trip{}, fmt.Errorf("begin trip creation: %w", err)
	}
	defer tx.Rollback(ctx)

	var plan subscriptions.Plan
	var startsAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT plan_id, starts_at FROM user_subscriptions
		WHERE user_id = $1 AND cancelled_at IS NULL AND ends_at > NOW()
		ORDER BY starts_at DESC LIMIT 1 FOR UPDATE
	`, riderUUID).Scan(&plan.ID, &startsAt)
	farePaise := baseFarePaise
	if err == nil {
		var active bool
		err = tx.QueryRow(ctx, `
			SELECT id, name, price_paise, monthly_ride_limit, discount_bps, active
			FROM subscription_plans WHERE id = $1
		`, plan.ID).Scan(&plan.ID, &plan.Name, &plan.PricePaise, &plan.MonthlyRideLimit, &plan.DiscountBPS, &active)
		if err != nil {
			return models.Trip{}, fmt.Errorf("load plan during trip creation: %w", err)
		}
		var ridesUsed int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM trip_requests
			WHERE rider_id = $1 AND status <> 'cancelled'
			  AND created_at >= GREATEST(date_trunc('month', NOW()), $2)
		`, riderUUID, startsAt).Scan(&ridesUsed); err != nil {
			return models.Trip{}, fmt.Errorf("count monthly subscription rides: %w", err)
		}
		if active {
			farePaise, err = subscriptions.ApplyDiscount(baseFarePaise, plan, ridesUsed)
			if err != nil {
				return models.Trip{}, fmt.Errorf("apply subscription discount to trip: %w", err)
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return models.Trip{}, fmt.Errorf("load active subscription during trip creation: %w", err)
	}

	var tripID, storedRiderID uuid.UUID
	var trip models.Trip
	err = tx.QueryRow(ctx, queries.Must("create_trip_with_party.sql"),
		riderUUID, strings.TrimSpace(originID), strings.TrimSpace(destination), distanceMeters, poolMode, farePaise, partySize, requiresAccessible,
		pickupLatitude, pickupLongitude, dropoffLatitude, dropoffLongitude,
	).Scan(&tripID, &storedRiderID, &trip.DriverID, &trip.OriginID, &trip.Destination,
		&trip.DistanceMeters, &trip.Status, &trip.PoolMode, &trip.PoolID, &trip.FareCents, &trip.PartySize,
		&trip.RequiresAccessible, &trip.CreatedAt, &trip.UpdatedAt, &trip.PickupLatitude, &trip.PickupLongitude,
		&trip.DropoffLatitude, &trip.DropoffLongitude)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Trip{}, ErrInvalidStation
		}
		return models.Trip{}, fmt.Errorf("insert trip request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Trip{}, fmt.Errorf("commit trip request: %w", err)
	}
	trip.ID, trip.RiderID = tripID.String(), storedRiderID.String()
	trip.FareAmount = float64(trip.FareCents) / 100
	return trip, nil
}
