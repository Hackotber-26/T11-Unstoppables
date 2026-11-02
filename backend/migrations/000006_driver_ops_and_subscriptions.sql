ALTER TABLE users ADD COLUMN phone TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN phone_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE trip_requests ADD COLUMN party_size INTEGER NOT NULL DEFAULT 1 CHECK (party_size BETWEEN 1 AND 8);
ALTER TABLE trip_requests ADD COLUMN requires_accessible BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE driver_locations (
    driver_id UUID NOT NULL REFERENCES driver_profiles(user_id) ON DELETE CASCADE,
    trip_id UUID NOT NULL REFERENCES trip_requests(id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude DOUBLE PRECISION NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    accuracy_meters DOUBLE PRECISION NOT NULL CHECK (accuracy_meters >= 0),
    speed_mps DOUBLE PRECISION CHECK (speed_mps IS NULL OR speed_mps >= 0),
    heading_degrees DOUBLE PRECISION CHECK (heading_degrees IS NULL OR heading_degrees >= 0 AND heading_degrees < 360),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (driver_id, trip_id)
);
CREATE INDEX driver_locations_trip_recorded_idx ON driver_locations (trip_id, recorded_at DESC);

CREATE TABLE driver_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES driver_profiles(user_id) ON DELETE CASCADE,
    document_type TEXT NOT NULL CHECK (document_type IN ('driving_license', 'vehicle_registration', 'insurance', 'identity')),
    file_name TEXT NOT NULL CHECK (length(file_name) BETWEEN 1 AND 180),
    content_type TEXT NOT NULL CHECK (content_type IN ('application/pdf', 'image/jpeg', 'image/png')),
    storage_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    review_note TEXT NOT NULL DEFAULT '' CHECK (length(review_note) <= 500),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);
CREATE INDEX driver_documents_driver_uploaded_idx ON driver_documents (driver_id, uploaded_at DESC);

CREATE TABLE station_bays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id TEXT NOT NULL REFERENCES stations(id),
    label TEXT NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    queue_length INTEGER NOT NULL DEFAULT 0 CHECK (queue_length >= 0),
    walking_meters DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (walking_meters >= 0),
    accessible BOOLEAN NOT NULL DEFAULT FALSE,
    is_open BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (station_id, label)
);

CREATE TABLE bay_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL UNIQUE REFERENCES trip_requests(id) ON DELETE CASCADE,
    bay_id UUID NOT NULL REFERENCES station_bays(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    arrived_at TIMESTAMPTZ
);

CREATE TABLE subscription_plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    price_paise BIGINT NOT NULL CHECK (price_paise >= 0),
    monthly_ride_limit INTEGER NOT NULL CHECK (monthly_ride_limit >= 0),
    discount_bps INTEGER NOT NULL CHECK (discount_bps BETWEEN 0 AND 10000),
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL REFERENCES subscription_plans(id),
    idempotency_key TEXT NOT NULL UNIQUE,
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ NOT NULL,
    cancelled_at TIMESTAMPTZ,
    CHECK (ends_at > starts_at)
);
CREATE UNIQUE INDEX user_one_active_subscription_idx ON user_subscriptions (user_id)
    WHERE cancelled_at IS NULL;
CREATE INDEX user_subscriptions_user_dates_idx ON user_subscriptions (user_id, starts_at DESC);

INSERT INTO subscription_plans (id, name, price_paise, monthly_ride_limit, discount_bps)
VALUES
    ('commuter-10', 'Commuter 10', 29900, 10, 1000),
    ('commuter-unlimited', 'Commuter Unlimited', 79900, 0, 2000)
ON CONFLICT (id) DO NOTHING;
