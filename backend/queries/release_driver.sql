UPDATE driver_profiles
SET status = 'available'
WHERE user_id = $1;