SELECT id, name, email, role, password_hash, created_at, phone, phone_verified
FROM users
WHERE LOWER(email) = LOWER($1);