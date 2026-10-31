CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'rider' CHECK (role IN ('rider', 'driver', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_lower_idx ON users (LOWER(email));

CREATE TABLE rider_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phone TEXT NOT NULL DEFAULT '',
    home_station_id TEXT
);

CREATE TABLE fleet_vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label TEXT NOT NULL UNIQUE,
    vehicle_type TEXT NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    accessibility_supported BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE driver_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    vehicle_id UUID REFERENCES fleet_vehicles(id),
    status TEXT NOT NULL DEFAULT 'offline' CHECK (status IN ('available', 'busy', 'offline')),
    verified BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE trip_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    origin_id TEXT NOT NULL,
    destination TEXT NOT NULL,
    driver_id UUID NOT NULL REFERENCES driver_profiles(user_id),
    status TEXT NOT NULL DEFAULT 'formed' CHECK (status IN ('formed', 'in_progress', 'completed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE trip_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id UUID NOT NULL REFERENCES rider_profiles(user_id),
    driver_id UUID REFERENCES driver_profiles(user_id),
    origin_id TEXT NOT NULL,
    destination TEXT NOT NULL,
    distance_meters DOUBLE PRECISION NOT NULL CHECK (distance_meters >= 0),
    status TEXT NOT NULL DEFAULT 'requested' CHECK (status IN ('requested', 'matched', 'in_progress', 'completed', 'cancelled')),
    pool_mode TEXT NOT NULL DEFAULT 'corridor' CHECK (pool_mode IN ('corridor', 'solo')),
    pool_id UUID REFERENCES trip_pools(id),
    fare_cents BIGINT NOT NULL DEFAULT 0 CHECK (fare_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX trips_match_queue_idx ON trip_requests (origin_id, status, created_at)
    WHERE status = 'requested';
CREATE INDEX trips_rider_created_idx ON trip_requests (rider_id, created_at DESC);

CREATE TABLE trip_pool_members (
    pool_id UUID NOT NULL REFERENCES trip_pools(id) ON DELETE CASCADE,
    trip_id UUID NOT NULL UNIQUE REFERENCES trip_requests(id),
    stop_sequence INTEGER NOT NULL CHECK (stop_sequence > 0),
    PRIMARY KEY (pool_id, trip_id)
);

CREATE TABLE wallet_accounts (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE wallet_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES wallet_accounts(user_id),
    trip_id UUID REFERENCES trip_requests(id),
    idempotency_key TEXT NOT NULL UNIQUE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents <> 0),
    entry_type TEXT NOT NULL CHECK (entry_type IN ('credit', 'debit', 'refund', 'adjustment')),
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX wallet_ledger_user_created_idx ON wallet_ledger_entries (user_id, created_at DESC);
CREATE UNIQUE INDEX wallet_trip_fare_once_idx ON wallet_ledger_entries (trip_id)
    WHERE entry_type = 'debit' AND trip_id IS NOT NULL;

CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_resource_created_idx ON audit_events (resource, resource_id, occurred_at DESC);
CREATE INDEX audit_actor_created_idx ON audit_events (actor_id, occurred_at DESC);