-- The machine-learning estimate of a project's success. The Python service
-- calculates it in the background, so a project may have no row yet.
-- One row per project: a new estimate replaces the old one.
CREATE TABLE ml_predictions (
    project_id     UUID        PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    probability_bp INTEGER     NOT NULL CHECK (probability_bp BETWEEN 0 AND 10000),
    model_version  TEXT        NOT NULL CHECK (char_length(model_version) BETWEEN 1 AND 50),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER ml_predictions_set_updated_at
    BEFORE UPDATE ON ml_predictions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();