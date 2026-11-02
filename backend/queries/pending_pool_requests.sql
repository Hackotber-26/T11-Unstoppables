SELECT id, rider_id, origin_id, destination, created_at,
    pickup_latitude, pickup_longitude, dropoff_latitude, dropoff_longitude, party_size
FROM trip_requests
WHERE status = 'requested' AND pool_mode = 'corridor'
    AND pickup_latitude IS NOT NULL AND pickup_longitude IS NOT NULL
    AND dropoff_latitude IS NOT NULL AND dropoff_longitude IS NOT NULL
    AND created_at >= NOW() - INTERVAL '5 minutes'
    AND created_at <= NOW() - INTERVAL '30 seconds'
ORDER BY origin_id, created_at
LIMIT 200
FOR UPDATE SKIP LOCKED;