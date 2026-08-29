-- name: GetUserByID :one
SELECT id, nickname, email, avatar FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, nickname, email, avatar FROM users
WHERE email = $1;
