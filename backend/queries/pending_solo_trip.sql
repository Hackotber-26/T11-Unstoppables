SELECT id, rider_id, origin_id, destination, created_at, distance_meters, pool_mode
FROM trip_requests
WHERE status = 'requested'
    AND (pool_mode = 'solo' OR (pool_mode = 'corridor' AND created_at < NOW() - INTERVAL '5 minutes'))
ORDER BY created_at
LIMIT 1
FOR UPDATE SKIP LOCKED;