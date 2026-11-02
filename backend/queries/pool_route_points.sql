WITH your_trip AS (
    SELECT t.id, t.pool_id, t.pickup_latitude, t.pickup_longitude
    FROM trip_requests t
    WHERE t.id = $1
      AND t.pool_id IS NOT NULL
      AND (
          $3 = 'admin'
          OR ($3 = 'rider' AND t.rider_id = $2)
          OR ($3 = 'driver' AND t.driver_id = $2)
      )
)
SELECT mine.pickup_latitude, mine.pickup_longitude,
       member.stop_sequence, trip.dropoff_latitude, trip.dropoff_longitude,
       trip.id = mine.id AS is_your_stop
FROM your_trip mine
JOIN trip_pool_members member ON member.pool_id = mine.pool_id
JOIN trip_requests trip ON trip.id = member.trip_id
WHERE trip.status IN ('matched', 'in_progress')
  AND trip.dropoff_latitude IS NOT NULL
  AND trip.dropoff_longitude IS NOT NULL
ORDER BY member.stop_sequence, trip.created_at;