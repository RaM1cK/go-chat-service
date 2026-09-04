-- name: ClaimDMChat :one
INSERT INTO dm_chats (user_a, user_b, chat_id)
VALUES ($1, $2, $3)
ON CONFLICT (user_a, user_b) DO NOTHING
RETURNING chat_id;

-- name: GetDMChat :one
SELECT chat_id FROM dm_chats
WHERE user_a = $1 AND user_b = $2;

-- name: CreateChat :one
INSERT INTO chats (type, name, logo)
VALUES ($1, $2, $3)
RETURNING id, type, name, logo;

-- name: AddUser :exec
INSERT INTO chat_members (chat_id, user_id)
VALUES ($1, $2);

-- name: RemoveUser :exec
DELETE FROM chat_members
WHERE chat_id = $2 AND user_id = $1;

-- name: DeleteChat :exec
DELETE FROM chat_members
WHERE chat_id = $1;

-- name: GetChat :one
SELECT id, type, name, logo FROM chats
WHERE id = $1;

-- name: GetChatsByUserId :many
SELECT c.id, c.type, c.name, c.logo
FROM chats c
JOIN chat_members cm ON cm.chat_id = c.id
WHERE cm.user_id = $1;

-- name: IsChatMember :one
SELECT EXISTS(
    SELECT 1 FROM chat_members
    WHERE chat_id = $1 AND user_id = $2
) AS is_member;
