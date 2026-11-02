SELECT u.id, u.name, COALESCE(v.id::text, ''), d.status, COALESCE(v.capacity, 0), d.verified
FROM users u
JOIN driver_profiles d ON d.user_id = u.id
LEFT JOIN fleet_vehicles v ON v.id = d.vehicle_id
WHERE u.id = $1;