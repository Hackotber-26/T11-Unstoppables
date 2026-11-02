SELECT u.id, u.name, u.email, r.phone, COALESCE(r.home_station_id, ''), u.created_at
FROM users u
JOIN rider_profiles r ON r.user_id = u.id
ORDER BY u.created_at DESC;