SELECT id, rider_id, COALESCE(driver_id::text, ''), origin_id, destination,
    distance_meters, status, pool_mode, COALESCE(pool_id::text, ''), fare_cents, created_at, updated_at
FROM trip_requests
WHERE ($1 = 'admin')
    OR ($1 = 'rider' AND rider_id = $2)
    OR ($1 = 'driver' AND driver_id = $2)
ORDER BY created_at DESC
LIMIT $3;