ALTER TABLE trip_requests
    ADD COLUMN pickup_latitude DOUBLE PRECISION,
    ADD COLUMN pickup_longitude DOUBLE PRECISION,
    ADD COLUMN dropoff_latitude DOUBLE PRECISION,
    ADD COLUMN dropoff_longitude DOUBLE PRECISION;

ALTER TABLE trip_requests
    ADD CONSTRAINT trip_requests_pickup_coordinates_check
        CHECK ((pickup_latitude IS NULL) = (pickup_longitude IS NULL)
            AND (pickup_latitude IS NULL OR pickup_latitude BETWEEN -90 AND 90)
            AND (pickup_longitude IS NULL OR pickup_longitude BETWEEN -180 AND 180)),
    ADD CONSTRAINT trip_requests_dropoff_coordinates_check
        CHECK ((dropoff_latitude IS NULL) = (dropoff_longitude IS NULL)
            AND (dropoff_latitude IS NULL OR dropoff_latitude BETWEEN -90 AND 90)
            AND (dropoff_longitude IS NULL OR dropoff_longitude BETWEEN -180 AND 180));

CREATE INDEX trip_requests_pool_route_idx
    ON trip_requests (pool_id, status, created_at)
    WHERE pool_id IS NOT NULL;