SELECT id, rider_id, COALESCE(driver_id::text, ''), origin_id, destination,
    distance_meters, status, pool_mode, COALESCE(pool_id::text, ''), fare_cents, created_at, updated_at
FROM trip_requests
WHERE id = $1;