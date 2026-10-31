SELECT id, name, email, role, password_hash, created_at
FROM users
WHERE LOWER(email) = LOWER($1);