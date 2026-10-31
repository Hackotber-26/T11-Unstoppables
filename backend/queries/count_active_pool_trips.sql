SELECT COUNT(*)
FROM trip_requests
WHERE pool_id = $1 AND status NOT IN ('completed', 'cancelled');