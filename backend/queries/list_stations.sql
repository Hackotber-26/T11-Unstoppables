SELECT s.id, s.name,
    COALESCE(array_agg(DISTINCT m.code ORDER BY m.code) FILTER (WHERE m.code IS NOT NULL), '{}')
FROM stations s
LEFT JOIN station_line_stops stop ON stop.station_id = s.id
LEFT JOIN metro_lines m ON m.code = stop.line_code
WHERE s.is_active
GROUP BY s.id, s.name
ORDER BY s.name;