CREATE TABLE IF NOT EXISTS tg_sessions (
    id TEXT PRIMARY KEY DEFAULT 'main',
    encrypted_data BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
