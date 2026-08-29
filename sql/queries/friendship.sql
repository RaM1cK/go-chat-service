-- name: CreateFriendship :exec
INSERT INTO friendships (user_a, user_b, status)
VALUES ($1, $2, $3)
ON CONFLICT (user_a, user_b) DO NOTHING;

-- name: GetFriendship :one
SELECT user_a, user_b, status, created_at FROM friendships
WHERE user_a = $1 AND user_b = $2;

-- name: UpdateFriendshipStatus :exec
UPDATE friendships
SET status = $3
WHERE user_a = $1 AND user_b = $2;

-- name: DeleteFriendship :exec
DELETE FROM friendships
WHERE user_a = $1 AND user_b = $2;
