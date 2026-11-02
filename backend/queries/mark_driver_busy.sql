UPDATE driver_profiles
SET status = 'busy'
WHERE user_id = $1;