SELECT COUNT(*),
    COUNT(*) FILTER (WHERE status = 'requested'),
    COUNT(*) FILTER (WHERE status = 'completed'),
    COUNT(*) FILTER (WHERE pool_mode = 'corridor'),
    (SELECT COUNT(*) FROM rider_profiles),
    (SELECT COUNT(DISTINCT rider_id) FROM trip_requests WHERE status IN ('requested', 'matched', 'in_progress')),
    (SELECT COUNT(*) FROM driver_profiles),
    (SELECT COUNT(*) FROM driver_profiles WHERE status = 'available'),
    (SELECT COUNT(*) FROM fleet_vehicles),
    (SELECT COUNT(*) FROM fleet_vehicles WHERE active),
    (SELECT COUNT(*) FROM stations WHERE is_active),
    (SELECT COUNT(*) FROM trip_pools WHERE status IN ('formed', 'in_progress'))
FROM trip_requests;