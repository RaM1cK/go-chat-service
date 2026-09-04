CREATE KEYSPACE IF NOT EXISTS messages
    WITH replication = {'class': 'NetworkTopologyStrategy', 'datacenter1': 1};

CREATE TABLE IF NOT EXISTS messages_by_chat (
    chat_id     UUID,
    created_at  TIMESTAMP,
    id          UUID,
    sender_id   UUID,
    data        TEXT,
    data_type   SMALLINT,
    quoted_id   UUID,
    PRIMARY KEY (chat_id, created_at, id)
) WITH CLUSTERING ORDER BY (created_at DESC, id DESC);