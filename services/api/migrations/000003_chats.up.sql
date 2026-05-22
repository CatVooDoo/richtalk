CREATE TYPE chat_type AS ENUM ('direct', 'group', 'channel');

CREATE TABLE chats (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type       chat_type   NOT NULL DEFAULT 'direct',
    -- NULL for direct chats; required for group/channel
    name       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chats_name_required_for_group
        CHECK (type = 'direct' OR name IS NOT NULL)
);

CREATE TRIGGER trg_chats_updated_at
    BEFORE UPDATE ON chats
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE chat_members (
    chat_id   UUID        NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX idx_chat_members_user_id ON chat_members (user_id);
-- chat_id is covered by the PK; explicit index for reverse lookup
CREATE INDEX idx_chat_members_chat_id ON chat_members (chat_id);

-- Guarantees at most one direct chat per ordered pair of users.
--
-- The CHECK constraint (user1_id < user2_id) enforces a canonical ordering
-- so that the pair (A,B) and (B,A) map to exactly one row.
-- Application must always insert with LEAST(a,b) as user1_id, GREATEST(a,b) as user2_id.
-- The UNIQUE constraint on (user1_id, user2_id) then prevents duplicates.
CREATE TABLE direct_chat_lookup (
    chat_id  UUID NOT NULL PRIMARY KEY REFERENCES chats(id) ON DELETE CASCADE,
    user1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    CONSTRAINT dcl_no_self_chat  CHECK (user1_id <> user2_id),
    CONSTRAINT dcl_canonical_order CHECK (user1_id < user2_id),
    CONSTRAINT dcl_unique_pair   UNIQUE (user1_id, user2_id)
);
