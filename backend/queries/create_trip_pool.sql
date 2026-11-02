INSERT INTO trip_pools (origin_id, destination, driver_id)
VALUES ($1, $2, $3)
RETURNING id;