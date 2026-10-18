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

// ReputationRepository reads the facts the reputation formula needs and
// stores snapshots and reviews. It embeds SwapRepository, so it also has
// GetSwap and GetUserByID.
type ReputationRepository struct {
	*SwapRepository
	pool *pgxpool.Pool
}

// NewReputationRepository creates the repository on top of the shared pool.
func NewReputationRepository(pool *pgxpool.Pool) *ReputationRepository {
	return &ReputationRepository{SwapRepository: NewSwapRepository(pool), pool: pool}
}

// GetReputationFacts collects everything the formula needs about one user.
func (r *ReputationRepository) GetReputationFacts(ctx context.Context, userID string) (domain.ReputationFacts, error) {
	var f domain.ReputationFacts

	err := r.pool.QueryRow(ctx, `SELECT created_at FROM users WHERE id = $1`, userID).Scan(&f.AccountSince)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReputationFacts{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ReputationFacts{}, fmt.Errorf("read user: %w", err)
	}

	// One pass over the user's swaps. FILTER counts only the rows that
	// match its condition, so one query gives all four numbers.
	err = r.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE s.status = 'completed'),
		       count(*) FILTER (WHERE s.status = 'failed'),
		       count(*) FILTER (WHERE s.status = 'cancelled' AND s.accepted_at IS NOT NULL),
		       COALESCE(sum(p.budget) FILTER (WHERE s.status IN ('completed', 'failed')), 0)::bigint
		FROM swaps s
		JOIN projects p ON p.id = s.project_id
		WHERE s.party_a_id = $1 OR s.party_b_id = $1`, userID,
	).Scan(&f.Completed, &f.Failed, &f.Cancelled, &f.TotalVolume)
	if err != nil {
		return domain.ReputationFacts{}, fmt.Errorf("read swap counts: %w", err)
	}

	// Every review of the user, with what we need to weigh its author.
	rows, err := r.pool.Query(ctx, `
		SELECT rv.rating, a.created_at,
		       (SELECT count(*) FROM swaps s
		        WHERE (s.party_a_id = a.id OR s.party_b_id = a.id) AND s.status = 'completed')
		FROM reviews rv
		JOIN users a ON a.id = rv.author_id
		WHERE rv.target_id = $1`, userID)
	if err != nil {
		return domain.ReputationFacts{}, fmt.Errorf("read reviews: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rf domain.ReviewFact
		if err := rows.Scan(&rf.Rating, &rf.AuthorSince, &rf.AuthorCompleted); err != nil {
			return domain.ReputationFacts{}, fmt.Errorf("read review: %w", err)
		}
		f.Reviews = append(f.Reviews, rf)
	}
	if err := rows.Err(); err != nil {
		return domain.ReputationFacts{}, fmt.Errorf("read reviews: %w", err)
	}
	return f, nil
}

// SaveReputation adds a new snapshot. Old snapshots are never changed (the
// table has an append-only trigger), so the full history stays available.
func (r *ReputationRepository) SaveReputation(ctx context.Context, userID string, rep domain.Reputation) (domain.Reputation, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO reputation_snapshots
		    (user_id, score_bp, completed_count, failed_count, cancelled_count, success_rate_bp,
		     avg_rating_x100, avg_budget, total_volume, trust_weight_bp, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at`,
		userID, rep.ScoreBP, rep.CompletedCount, rep.FailedCount, rep.CancelledCount, rep.SuccessRateBP,
		rep.AvgRatingX100, rep.AvgBudget, rep.TotalVolume, rep.TrustWeightBP, rep.Reason,
	).Scan(&rep.CreatedAt)
	if err != nil {
		return domain.Reputation{}, fmt.Errorf("save reputation: %w", err)
	}
	return rep, nil
}

const reputationColumns = `
	score_bp, completed_count, failed_count, cancelled_count, success_rate_bp,
	avg_rating_x100, avg_budget, total_volume, trust_weight_bp, reason, created_at`

func scanReputation(row pgx.Row) (domain.Reputation, error) {
	var rep domain.Reputation
	err := row.Scan(&rep.ScoreBP, &rep.CompletedCount, &rep.FailedCount, &rep.CancelledCount,
		&rep.SuccessRateBP, &rep.AvgRatingX100, &rep.AvgBudget, &rep.TotalVolume,
		&rep.TrustWeightBP, &rep.Reason, &rep.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reputation{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Reputation{}, fmt.Errorf("read reputation: %w", err)
	}
	return rep, nil
}

// GetLatestReputation returns the current snapshot of a user.
func (r *ReputationRepository) GetLatestReputation(ctx context.Context, userID string) (domain.Reputation, error) {
	return scanReputation(r.pool.QueryRow(ctx, `SELECT `+reputationColumns+`
		FROM reputation_snapshots WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT 1`, userID))
}

// ListReputationHistory returns the last snapshots, oldest first (ready
// to be drawn as a line chart).
func (r *ReputationRepository) ListReputationHistory(ctx context.Context, userID string, limit int) ([]domain.Reputation, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+reputationColumns+` FROM (
		SELECT * FROM reputation_snapshots WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2
	) recent ORDER BY created_at, id`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list reputation: %w", err)
	}
	defer rows.Close()

	history := []domain.Reputation{}
	for rows.Next() {
		rep, err := scanReputation(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list reputation: %w", err)
	}
	return history, nil
}

// CreateReview saves a review. The table allows one review per author per
// swap, so a second one becomes a clear 409.
func (r *ReputationRepository) CreateReview(ctx context.Context, rv domain.Review) (domain.Review, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO reviews (swap_id, author_id, target_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		rv.SwapID, rv.AuthorID, rv.TargetID, rv.Rating, rv.Comment,
	).Scan(&rv.ID, &rv.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "reviews_one_per_author_per_swap" {
		return domain.Review{}, &domain.ConflictError{Message: "you already reviewed this swap"}
	}
	if err != nil {
		return domain.Review{}, fmt.Errorf("insert review: %w", err)
	}
	return rv, nil
}

// ListReviews returns the newest reviews about a user.
func (r *ReputationRepository) ListReviews(ctx context.Context, targetID string, limit int) ([]domain.Review, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rv.id, rv.swap_id, rv.author_id, a.username, rv.target_id, rv.rating, rv.comment, rv.created_at
		FROM reviews rv
		JOIN users a ON a.id = rv.author_id
		WHERE rv.target_id = $1
		ORDER BY rv.created_at DESC, rv.id
		LIMIT $2`, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	reviews := []domain.Review{}
	for rows.Next() {
		var rv domain.Review
		if err := rows.Scan(&rv.ID, &rv.SwapID, &rv.AuthorID, &rv.AuthorUsername, &rv.TargetID,
			&rv.Rating, &rv.Comment, &rv.CreatedAt); err != nil {
			return nil, fmt.Errorf("read review: %w", err)
		}
		reviews = append(reviews, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	return reviews, nil
}
