SELECT COUNT(*),
    COUNT(*) FILTER (WHERE status = 'requested'),
    COUNT(*) FILTER (WHERE status = 'completed'),
    COUNT(*) FILTER (WHERE pool_mode = 'corridor')
FROM trip_requests;