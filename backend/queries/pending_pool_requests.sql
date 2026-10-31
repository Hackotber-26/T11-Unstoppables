SELECT id, rider_id, origin_id, destination, created_at
FROM trip_requests
WHERE status = 'requested' AND pool_mode = 'corridor'
    AND created_at >= NOW() - INTERVAL '5 minutes'
    AND created_at <= NOW() - INTERVAL '30 seconds'
ORDER BY origin_id, destination, created_at
LIMIT 200
FOR UPDATE SKIP LOCKED;