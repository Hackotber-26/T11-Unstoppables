UPDATE driver_profiles d
SET status = $2
WHERE d.user_id = $1
    AND d.verified
    AND NOT EXISTS (
        SELECT 1 FROM trip_requests t
        WHERE t.driver_id = d.user_id AND t.status = 'in_progress'
    )
RETURNING d.user_id,
    (SELECT u.name FROM users u WHERE u.id = d.user_id),
    COALESCE(d.vehicle_id::text, ''),
    d.status,
    COALESCE((SELECT v.capacity FROM fleet_vehicles v WHERE v.id = d.vehicle_id), 0),
    d.verified;