CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      CITEXT      NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_username_length CHECK (char_length(username) BETWEEN 3 AND 32),
    -- Only alphanumeric + underscore, must not start with digit
    CONSTRAINT users_username_format CHECK (username ~ '^[a-zA-Z0-9_]+$')
);

CREATE INDEX idx_users_username ON users (username);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Refresh tokens are opaque random strings stored as bcrypt hashes.
-- One row per active session; expired rows are cleaned up periodically.
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT refresh_tokens_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX idx_refresh_tokens_user_id  ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires  ON refresh_tokens (expires_at);
