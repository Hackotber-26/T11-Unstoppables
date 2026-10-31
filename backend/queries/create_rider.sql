INSERT INTO users (name, email, password_hash, role)
VALUES ($1, LOWER($2), $3, 'rider')
RETURNING id, name, email, role, created_at;