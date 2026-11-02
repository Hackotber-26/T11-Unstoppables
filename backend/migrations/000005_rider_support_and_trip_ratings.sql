CREATE TABLE support_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('trip', 'wallet', 'safety', 'other')),
    subject TEXT NOT NULL CHECK (length(subject) BETWEEN 1 AND 120),
    message TEXT NOT NULL CHECK (length(message) BETWEEN 1 AND 2000),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX support_tickets_user_created_idx ON support_tickets (user_id, created_at DESC);
CREATE INDEX support_tickets_status_created_idx ON support_tickets (status, created_at DESC);

CREATE TABLE trip_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID NOT NULL UNIQUE REFERENCES trip_requests(id) ON DELETE CASCADE,
    rider_id UUID NOT NULL REFERENCES rider_profiles(user_id),
    driver_id UUID NOT NULL REFERENCES driver_profiles(user_id),
    stars SMALLINT NOT NULL CHECK (stars BETWEEN 1 AND 5),
    comment TEXT NOT NULL DEFAULT '' CHECK (length(comment) <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX trip_ratings_driver_created_idx ON trip_ratings (driver_id, created_at DESC);
