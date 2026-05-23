ALTER TABLE messages
  ADD COLUMN attachment_type TEXT,
  ADD COLUMN attachment_url  TEXT,
  ADD COLUMN attachment_name TEXT,
  ADD COLUMN attachment_size BIGINT;

ALTER TABLE messages DROP CONSTRAINT messages_content_not_empty;
ALTER TABLE messages ADD CONSTRAINT messages_content_or_attachment
  CHECK (char_length(content) > 0 OR attachment_url IS NOT NULL);
