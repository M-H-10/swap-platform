package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// PredictionRepository stores the ML estimates. It embeds ProjectRepository,
// so it also has GetProject and GetCreatorStats.
type PredictionRepository struct {
	*ProjectRepository
	pool *pgxpool.Pool
}

// NewPredictionRepository creates the repository on top of the shared pool.
func NewPredictionRepository(pool *pgxpool.Pool) *PredictionRepository {
	return &PredictionRepository{ProjectRepository: NewProjectRepository(pool), pool: pool}
}

// SavePrediction inserts the estimate, or replaces the old one of the same
// project ("upsert": INSERT ... ON CONFLICT DO UPDATE).
func (r *PredictionRepository) SavePrediction(ctx context.Context, res domain.PredictionResult) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ml_predictions (project_id, probability_bp, model_version)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id) DO UPDATE
		SET probability_bp = EXCLUDED.probability_bp,
		    model_version  = EXCLUDED.model_version`,
		res.ProjectID, res.ProbabilityBP, res.ModelVersion)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign key: no such project
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("save prediction: %w", err)
	}
	return nil
}

// GetPrediction returns the estimate of a project, or ErrNotFound when the
// ML service has not answered yet.
func (r *PredictionRepository) GetPrediction(ctx context.Context, projectID string) (domain.Prediction, error) {
	var p domain.Prediction
	err := r.pool.QueryRow(ctx, `
		SELECT project_id, probability_bp, model_version, updated_at
		FROM ml_predictions WHERE project_id = $1`, projectID,
	).Scan(&p.ProjectID, &p.ProbabilityBP, &p.ModelVersion, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Prediction{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Prediction{}, fmt.Errorf("read prediction: %w", err)
	}
	return p, nil
}
