-- Add columns for encrypted context and its nonce
ALTER TABLE messages ADD COLUMN IF NOT EXISTS encrypted_context BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS nonce_context BYTEA;
