CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS channels (
    id BIGINT PRIMARY KEY,
    title TEXT NOT NULL,
    username TEXT
);

CREATE TABLE IF NOT EXISTS messages (
    id BIGINT,
    channel_id BIGINT,
    message TEXT,
    context_text TEXT,
    date TIMESTAMP,
    embedding vector(768),
    PRIMARY KEY(id, channel_id),
    FOREIGN KEY(channel_id) REFERENCES channels(id)
);
