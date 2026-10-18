-- Reviews and the reputation history.

CREATE TABLE reviews (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    swap_id    UUID        NOT NULL REFERENCES swaps (id) ON DELETE RESTRICT,
    author_id  UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    target_id  UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    rating     SMALLINT    NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT        NOT NULL DEFAULT '' CHECK (char_length(comment) <= 1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT reviews_one_per_author_per_swap UNIQUE (swap_id, author_id),
    CONSTRAINT reviews_not_self CHECK (author_id <> target_id)
);

CREATE INDEX reviews_target_id_idx ON reviews (target_id);

-- Every recalculation of a user's reputation adds a new row. The latest
-- row is the current reputation; the older rows draw the history chart.
CREATE TABLE reputation_snapshots (
    id              BIGSERIAL   PRIMARY KEY,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    score_bp        INTEGER     NOT NULL CHECK (score_bp BETWEEN 0 AND 10000),
    completed_count INTEGER     NOT NULL DEFAULT 0 CHECK (completed_count >= 0),
    failed_count    INTEGER     NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    cancelled_count INTEGER     NOT NULL DEFAULT 0 CHECK (cancelled_count >= 0),
    disputes_count  INTEGER     NOT NULL DEFAULT 0 CHECK (disputes_count >= 0),
    success_rate_bp INTEGER     NOT NULL DEFAULT 0 CHECK (success_rate_bp BETWEEN 0 AND 10000),
    avg_rating_x100 INTEGER     CHECK (avg_rating_x100 BETWEEN 100 AND 500),
    avg_delay_days  INTEGER     NOT NULL DEFAULT 0,
    avg_budget      BIGINT      NOT NULL DEFAULT 0 CHECK (avg_budget >= 0),
    total_volume    BIGINT      NOT NULL DEFAULT 0 CHECK (total_volume >= 0),
    trust_weight_bp INTEGER     NOT NULL DEFAULT 10000 CHECK (trust_weight_bp BETWEEN 0 AND 10000),
    reason          TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reputation_snapshots_user_id_created_at_idx
    ON reputation_snapshots (user_id, created_at DESC);

CREATE TRIGGER reputation_snapshots_append_only
    BEFORE UPDATE OR DELETE ON reputation_snapshots
    FOR EACH ROW EXECUTE FUNCTION forbid_change();