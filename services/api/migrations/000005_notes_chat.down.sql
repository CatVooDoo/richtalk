ALTER TABLE messages DROP COLUMN IF EXISTS attachment_size;
ALTER TABLE messages DROP COLUMN IF EXISTS attachment_name;
ALTER TABLE messages DROP COLUMN IF EXISTS attachment_url;
ALTER TABLE messages DROP COLUMN IF EXISTS attachment_type;

-- Restore original constraint
ALTER TABLE chats DROP CONSTRAINT IF EXISTS chats_name_required_for_group;
ALTER TABLE chats ADD CONSTRAINT chats_name_required_for_group
    CHECK (type = 'direct' OR name IS NOT NULL);

-- Note: removing enum values is not supported in PostgreSQL.
-- The 'notes' value will remain in the chat_type enum.
