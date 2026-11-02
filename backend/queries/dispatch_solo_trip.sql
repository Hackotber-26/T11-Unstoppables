UPDATE trip_requests
SET status = 'matched', driver_id = $2, pool_mode = $3,
    fare_paise = CASE WHEN $4 THEN $5 ELSE fare_paise END, updated_at = NOW()
WHERE id = $1;