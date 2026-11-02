INSERT INTO trip_requests (rider_id, origin_id, destination, distance_meters, pool_mode, fare_paise)
SELECT $1, stations.id, $3, $4, $5, $6
FROM stations
WHERE stations.id = $2 AND stations.is_active
RETURNING id, rider_id, COALESCE(driver_id::text, ''), origin_id, destination,
    distance_meters, status, pool_mode, COALESCE(pool_id::text, ''), fare_paise, party_size,
    requires_accessible, created_at, updated_at;