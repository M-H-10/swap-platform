-- Projects, swap contracts, escrow holds and the money ledger.
-- Money columns are BIGINT kopecks. Percentages are INTEGER basis
-- points (bp): 10000 bp = 100%, 6250 bp = 62.50%.

CREATE TABLE projects (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id             UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title                  TEXT        NOT NULL CHECK (char_length(title) BETWEEN 3 AND 120),
    description            TEXT        NOT NULL CHECK (char_length(description) BETWEEN 10 AND 5000),
    category               TEXT        NOT NULL
                                       CHECK (category IN ('web', 'mobile', 'design', 'marketing', 'other')),
    budget                 BIGINT      NOT NULL CHECK (budget > 0),
    duration_days          INTEGER     NOT NULL CHECK (duration_days BETWEEN 1 AND 3650),
    complexity             SMALLINT    NOT NULL CHECK (complexity BETWEEN 1 AND 10),
    team_size              SMALLINT    NOT NULL CHECK (team_size BETWEEN 1 AND 100),
    kpis                   JSONB       NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(kpis) = 'array'),
    success_probability_bp INTEGER     NOT NULL CHECK (success_probability_bp BETWEEN 0 AND 10000),
    status                 TEXT        NOT NULL DEFAULT 'open'
                                       CHECK (status IN ('open', 'in_progress', 'completed', 'failed', 'cancelled')),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX projects_creator_id_idx ON projects (creator_id);
CREATE INDEX projects_status_created_at_idx ON projects (status, created_at DESC);

CREATE TRIGGER projects_set_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE swaps (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id             UUID        NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    party_a_id             UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    party_b_id             UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    party_a_role           TEXT        NOT NULL,
    party_b_role           TEXT        NOT NULL,
    share_a_bp             INTEGER     NOT NULL CHECK (share_a_bp BETWEEN 0 AND 10000),
    share_b_bp             INTEGER     NOT NULL CHECK (share_b_bp BETWEEN 0 AND 10000),
    penalty_a              BIGINT      NOT NULL CHECK (penalty_a >= 0),
    penalty_b              BIGINT      NOT NULL CHECK (penalty_b >= 0),
    escrow_amount          BIGINT      NOT NULL CHECK (escrow_amount > 0),
    success_probability_bp INTEGER     NOT NULL CHECK (success_probability_bp BETWEEN 0 AND 10000),
    risk_a                 JSONB       NOT NULL DEFAULT '{}',
    risk_b                 JSONB       NOT NULL DEFAULT '{}',
    status                 TEXT        NOT NULL DEFAULT 'proposed'
                                       CHECK (status IN ('proposed', 'active', 'completed', 'failed', 'cancelled', 'rejected')),
    outcome_a              TEXT        CHECK (outcome_a IN ('success', 'failure')),
    outcome_b              TEXT        CHECK (outcome_b IN ('success', 'failure')),
    accepted_at            TIMESTAMPTZ,
    deadline_at            TIMESTAMPTZ,
    closed_at              TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT swaps_different_parties CHECK (party_a_id <> party_b_id),
    CONSTRAINT swaps_shares_sum        CHECK (share_a_bp + share_b_bp = 10000),
    CONSTRAINT swaps_penalties_sum     CHECK (penalty_a + penalty_b = escrow_amount)
);

CREATE INDEX swaps_project_id_idx ON swaps (project_id);
CREATE INDEX swaps_party_a_id_idx ON swaps (party_a_id);
CREATE INDEX swaps_party_b_id_idx ON swaps (party_b_id);

-- A project can have many proposals, but only one active swap at a time.
CREATE UNIQUE INDEX swaps_one_active_per_project ON swaps (project_id) WHERE status = 'active';

CREATE TRIGGER swaps_set_updated_at
    BEFORE UPDATE ON swaps
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Money each party has locked for a given swap.
CREATE TABLE escrow_holds (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    swap_id    UUID        NOT NULL REFERENCES swaps (id) ON DELETE RESTRICT,
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    amount     BIGINT      NOT NULL CHECK (amount > 0),
    status     TEXT        NOT NULL DEFAULT 'locked'
                           CHECK (status IN ('locked', 'released', 'forfeited')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at  TIMESTAMPTZ,

    CONSTRAINT escrow_holds_one_per_party UNIQUE (swap_id, user_id)
);

-- The ledger: one row for every money movement. Rows are never changed
-- or deleted, so the full history can always be audited.
CREATE TABLE transactions (
    id            BIGSERIAL   PRIMARY KEY,
    user_id       UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    swap_id       UUID        REFERENCES swaps (id) ON DELETE RESTRICT,
    type          TEXT        NOT NULL
                              CHECK (type IN ('deposit', 'escrow_lock', 'escrow_release',
                                              'penalty_paid', 'penalty_received', 'fee')),
    amount        BIGINT      NOT NULL CHECK (amount > 0),
    balance_after BIGINT      NOT NULL CHECK (balance_after >= 0),
    frozen_after  BIGINT      NOT NULL CHECK (frozen_after >= 0),
    description   TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX transactions_user_id_created_at_idx ON transactions (user_id, created_at DESC);
CREATE INDEX transactions_swap_id_idx ON transactions (swap_id);

CREATE TRIGGER transactions_append_only
    BEFORE UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE FUNCTION forbid_change();