UPDATE trip_pools
SET status = 'completed'
WHERE id = $1;