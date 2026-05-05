-- name: UserUpdatedAt :one
SELECT updated_at FROM users WHERE id = $1;

-- name: UserByName :one
SELECT id, password FROM users WHERE username = $1;
