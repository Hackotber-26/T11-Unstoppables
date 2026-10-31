WITH current_trip AS (
    SELECT origin_id, destination, distance_meters, created_at
    FROM trip_requests
    WHERE rider_id = $1 AND status = 'requested'
    ORDER BY created_at DESC
    LIMIT 1
)
SELECT t.rider_id, t.destination, t.distance_meters, t.created_at,
    c.destination, c.distance_meters, c.created_at
FROM trip_requests t
CROSS JOIN current_trip c
WHERE t.rider_id <> $1 AND t.status = 'requested'
    AND t.origin_id = c.origin_id
    AND ABS(EXTRACT(EPOCH FROM (t.created_at - c.created_at))) <= 300
ORDER BY t.created_at ASC
LIMIT 100;