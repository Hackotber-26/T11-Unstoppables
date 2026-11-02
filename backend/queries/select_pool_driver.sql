SELECT d.user_id, d.verified, v.verified
FROM driver_profiles d
JOIN fleet_vehicles v ON v.id = d.vehicle_id
WHERE d.status = 'available' AND d.verified = TRUE
    AND v.active = TRUE AND v.verified = TRUE AND v.capacity >= $1
ORDER BY d.user_id
LIMIT 1
FOR UPDATE OF d SKIP LOCKED;