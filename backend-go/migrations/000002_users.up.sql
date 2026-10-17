-- Accounts, login sessions and wallets.

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL CHECK (char_length(username) BETWEEN 3 AND 32),
    email         TEXT        NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254),
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'developer'
                              CHECK (role IN ('developer', 'designer', 'manager', 'other')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness: "Ali" and "ali" are the same user.
CREATE UNIQUE INDEX users_username_lower_key ON users (lower(username));
CREATE UNIQUE INDEX users_email_lower_key    ON users (lower(email));

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- One row per login session. Only a hash of the token is stored,
-- so a leaked database does not leak usable tokens.
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

-- Money is stored in kopecks (BIGINT), never as a fraction.
-- balance        = money the user can use
-- frozen_balance = money locked in escrow for active swaps
CREATE TABLE wallets (
    user_id        UUID        PRIMARY KEY REFERENCES users (id) ON DELETE RESTRICT,
    balance        BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0),
    frozen_balance BIGINT      NOT NULL DEFAULT 0 CHECK (frozen_balance >= 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER wallets_set_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();