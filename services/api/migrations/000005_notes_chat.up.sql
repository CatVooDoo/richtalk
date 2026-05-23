-- Add 'notes' to chat_type enum (PG16: safe in transaction)
ALTER TYPE chat_type ADD VALUE IF NOT EXISTS 'notes';

-- Update check to allow notes chats without a name
ALTER TABLE chats DROP CONSTRAINT IF EXISTS chats_name_required_for_group;
ALTER TABLE chats ADD CONSTRAINT chats_name_required_for_group
    CHECK (type IN ('direct', 'notes') OR name IS NOT NULL);

-- Attachment columns (architecture-only in MVP; no upload logic yet)
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_type VARCHAR(20) NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_url  TEXT NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_name TEXT NULL;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_size BIGINT NULL;
