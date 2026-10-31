UPDATE trip_requests
SET status = 'matched', driver_id = $2, pool_id = $3, updated_at = NOW()
WHERE id = $1 AND status = 'requested';