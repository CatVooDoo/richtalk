ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_type TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_url  TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_name TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_size BIGINT;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_content_not_empty;
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_content_or_attachment;
ALTER TABLE messages ADD CONSTRAINT messages_content_or_attachment
  CHECK (char_length(content) > 0 OR attachment_url IS NOT NULL);
