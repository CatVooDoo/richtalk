CREATE TABLE messages (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id    UUID        NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    author_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Soft delete: NULL = alive, non-NULL = deleted. Content is preserved
    -- in the row but the API returns a placeholder "message deleted" when
    -- deleted_at IS NOT NULL.
    deleted_at TIMESTAMPTZ,

    CONSTRAINT messages_content_not_empty CHECK (char_length(content) > 0)
);

-- Primary access pattern: paginate messages in a chat newest-first.
-- Supports cursor pagination: WHERE chat_id = $1 AND created_at < $cursor
CREATE INDEX idx_messages_chat_created ON messages (chat_id, created_at DESC);

CREATE INDEX idx_messages_author_id ON messages (author_id);

CREATE TRIGGER trg_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
