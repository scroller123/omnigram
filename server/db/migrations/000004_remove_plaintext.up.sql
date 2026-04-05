-- Step 1: Remove plaintext columns from messages table
ALTER TABLE messages DROP COLUMN IF EXISTS message;
ALTER TABLE messages DROP COLUMN IF EXISTS context_text;

-- Ensure encrypted_data and nonce are NOT NULL for future entries
ALTER TABLE messages ALTER COLUMN encrypted_data SET NOT NULL;
ALTER TABLE messages ALTER COLUMN nonce SET NOT NULL;
