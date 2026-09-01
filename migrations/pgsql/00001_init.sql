-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id       UUID PRIMARY KEY
);

CREATE TABLE chats (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type smallint NOT NULL DEFAULT 0,
    name text,
    logo text
);

CREATE TABLE chat_members (
    chat_id   UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);
CREATE INDEX chat_members_user_idx ON chat_members (user_id);

CREATE TABLE dm_chats (
    user_a  UUID NOT NULL REFERENCES users(id),
    user_b  UUID NOT NULL REFERENCES users(id),
    chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    PRIMARY KEY (user_a, user_b),
    CHECK (user_a < user_b)
);

CREATE TABLE friendships (
    user_a     UUID NOT NULL REFERENCES users(id),
    user_b     UUID NOT NULL REFERENCES users(id),
    status     smallint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_a, user_b),
    CHECK (user_a < user_b)
);
CREATE INDEX friendships_user_b_idx ON friendships (user_b);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS friendships;
DROP TABLE IF EXISTS dm_chats;
DROP TABLE IF EXISTS chat_members;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
