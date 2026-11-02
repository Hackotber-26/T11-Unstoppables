SELECT rider_id, driver_id, status, pool_id
FROM trip_requests
WHERE id = $1
FOR UPDATE;