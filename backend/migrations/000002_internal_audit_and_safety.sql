ALTER TABLE fleet_vehicles ADD COLUMN IF NOT EXISTS verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE fleet_vehicles ADD COLUMN IF NOT EXISTS accessibility_supported BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE driver_profiles ADD COLUMN IF NOT EXISTS verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE driver_profiles ALTER COLUMN status SET DEFAULT 'offline';

CREATE UNIQUE INDEX IF NOT EXISTS wallet_trip_fare_once_idx ON wallet_ledger_entries (trip_id)
    WHERE entry_type = 'debit' AND trip_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS audit_resource_created_idx ON audit_events (resource, resource_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS audit_actor_created_idx ON audit_events (actor_id, occurred_at DESC);