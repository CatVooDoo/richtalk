ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_content_or_attachment;
ALTER TABLE messages ADD CONSTRAINT messages_content_not_empty CHECK (char_length(content) > 0);

ALTER TABLE messages
  DROP COLUMN IF EXISTS attachment_type,
  DROP COLUMN IF EXISTS attachment_url,
  DROP COLUMN IF EXISTS attachment_name,
  DROP COLUMN IF EXISTS attachment_size;
