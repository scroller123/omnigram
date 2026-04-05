-- Step 1: Add enrichment tracking table
CREATE TABLE IF NOT EXISTS enrichment_tasks (
    id SERIAL PRIMARY KEY,
    dialog_id BIGINT NOT NULL,
    session_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    total_messages INT DEFAULT 0,
    processed_messages INT DEFAULT 0,
    last_message_id INT DEFAULT 0,
    error_message TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(dialog_id, session_id)
);

-- Step 2: Add encryption columns to messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS encrypted_data BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS nonce BYTEA;
