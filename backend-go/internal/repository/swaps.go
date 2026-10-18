package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// SwapRepository reads and writes swaps. It embeds the project and user
// repositories, so it also has GetProject, GetCreatorStats and GetUserByID
// without writing them again.
type SwapRepository struct {
	*ProjectRepository
	*UserRepository
	pool *pgxpool.Pool
}

// NewSwapRepository creates the repository on top of the shared pool.
func NewSwapRepository(pool *pgxpool.Pool) *SwapRepository {
	return &SwapRepository{
		ProjectRepository: NewProjectRepository(pool),
		UserRepository:    NewUserRepository(pool),
		pool:              pool,
	}
}

// swapColumns and swapFrom are shared by every query that reads swaps.
const swapColumns = `
	s.id, s.project_id, p.title, s.party_a_id, ua.username, s.party_b_id, ub.username,
	s.party_a_role, s.party_b_role, s.share_a_bp, s.share_b_bp, s.escrow_amount,
	s.penalty_a, s.penalty_b, s.success_probability_bp, s.risk_a, s.risk_b,
	s.status, s.outcome_a, s.outcome_b, s.cancel_requested_by,
	s.accepted_at, s.deadline_at, s.closed_at, s.created_at, s.updated_at`

const swapFrom = `
	FROM swaps s
	JOIN projects p ON p.id = s.project_id
	JOIN users ua ON ua.id = s.party_a_id
	JOIN users ub ON ub.id = s.party_b_id`

func scanSwap(row pgx.Row) (domain.Swap, error) {
	var s domain.Swap
	var riskA, riskB []byte
	err := row.Scan(&s.ID, &s.ProjectID, &s.ProjectTitle, &s.PartyAID, &s.PartyAUsername,
		&s.PartyBID, &s.PartyBUsername, &s.PartyARole, &s.PartyBRole,
		&s.Terms.ShareABP, &s.Terms.ShareBBP, &s.Terms.EscrowAmount,
		&s.Terms.PenaltyA, &s.Terms.PenaltyB, &s.Terms.SuccessProbabilityBP, &riskA, &riskB,
		&s.Status, &s.OutcomeA, &s.OutcomeB, &s.CancelRequestedBy,
		&s.AcceptedAt, &s.DeadlineAt, &s.ClosedAt, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Swap{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Swap{}, fmt.Errorf("read swap: %w", err)
	}
	if err := json.Unmarshal(riskA, &s.Terms.RiskA); err != nil {
		return domain.Swap{}, fmt.Errorf("read risk_a: %w", err)
	}
	if err := json.Unmarshal(riskB, &s.Terms.RiskB); err != nil {
		return domain.Swap{}, fmt.Errorf("read risk_b: %w", err)
	}
	return s, nil
}

// CreateSwap saves a new proposal.
func (r *SwapRepository) CreateSwap(ctx context.Context, s domain.Swap) (domain.Swap, error) {
	riskA, _ := json.Marshal(s.Terms.RiskA)
	riskB, _ := json.Marshal(s.Terms.RiskB)

	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO swaps (project_id, party_a_id, party_b_id, party_a_role, party_b_role,
		                   share_a_bp, share_b_bp, penalty_a, penalty_b, escrow_amount,
		                   success_probability_bp, risk_a, risk_b)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`,
		s.ProjectID, s.PartyAID, s.PartyBID, s.PartyARole, s.PartyBRole,
		s.Terms.ShareABP, s.Terms.ShareBBP, s.Terms.PenaltyA, s.Terms.PenaltyB, s.Terms.EscrowAmount,
		s.Terms.SuccessProbabilityBP, riskA, riskB,
	).Scan(&id)
	if err != nil {
		return domain.Swap{}, fmt.Errorf("insert swap: %w", err)
	}
	return r.GetSwap(ctx, id)
}

// GetSwap finds one swap by ID.
func (r *SwapRepository) GetSwap(ctx context.Context, id string) (domain.Swap, error) {
	return scanSwap(r.pool.QueryRow(ctx, `SELECT `+swapColumns+swapFrom+` WHERE s.id = $1`, id))
}

// ListSwaps returns the swaps where the user is party A or B, newest first.
// An empty status means every status.
func (r *SwapRepository) ListSwaps(ctx context.Context, userID string, status domain.SwapStatus) ([]domain.Swap, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+swapColumns+swapFrom+`
		WHERE (s.party_a_id = $1 OR s.party_b_id = $1)
		  AND ($2 = '' OR s.status = $2)
		ORDER BY s.created_at DESC, s.id
		LIMIT 100`, userID, string(status))
	if err != nil {
		return nil, fmt.Errorf("list swaps: %w", err)
	}
	defer rows.Close()

	swaps := []domain.Swap{}
	for rows.Next() {
		s, err := scanSwap(rows)
		if err != nil {
			return nil, err
		}
		swaps = append(swaps, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list swaps: %w", err)
	}
	return swaps, nil
}

// InTx runs fn inside one database transaction. If fn returns an error,
// everything it did is rolled back; otherwise it is committed.
func (r *SwapRepository) InTx(ctx context.Context, fn func(tx domain.SwapTx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(&swapTx{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// swapTx implements domain.SwapTx with one open PostgreSQL transaction.
type swapTx struct {
	tx pgx.Tx
}

// LockSwap reads a swap and locks its row ("FOR UPDATE OF s": only the
// swaps row, not the joined projects and users rows).
func (t *swapTx) LockSwap(ctx context.Context, id string) (domain.Swap, error) {
	return scanSwap(t.tx.QueryRow(ctx, `SELECT `+swapColumns+swapFrom+` WHERE s.id = $1 FOR UPDATE OF s`, id))
}

// LockProject reads a project and locks its row.
func (t *swapTx) LockProject(ctx context.Context, id string) (domain.Project, error) {
	return scanProject(t.tx.QueryRow(ctx, `SELECT `+projectColumns+projectFrom+` WHERE p.id = $1 FOR UPDATE OF p`, id))
}

// LockWallets locks wallets in a fixed order (sorted by user_id). If two
// transactions lock the same two wallets in opposite orders, each waits
// for the other forever (a deadlock); a fixed order prevents that.
func (t *swapTx) LockWallets(ctx context.Context, userIDs ...string) error {
	_, err := t.tx.Exec(ctx, `
		SELECT user_id FROM wallets
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id
		FOR UPDATE`, userIDs)
	if err != nil {
		return fmt.Errorf("lock wallets: %w", err)
	}
	return nil
}

func (t *swapTx) MoveMoney(ctx context.Context, m domain.MoneyMove) error {
	return moveMoney(ctx, t.tx, m)
}

func (t *swapTx) CreateHold(ctx context.Context, swapID, userID string, amount int64) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO escrow_holds (swap_id, user_id, amount) VALUES ($1, $2, $3)`,
		swapID, userID, amount)
	if err != nil {
		return fmt.Errorf("create hold: %w", err)
	}
	return nil
}

func (t *swapTx) CloseHold(ctx context.Context, swapID, userID string, status domain.HoldStatus) error {
	tag, err := t.tx.Exec(ctx, `
		UPDATE escrow_holds SET status = $3, closed_at = now()
		WHERE swap_id = $1 AND user_id = $2 AND status = 'locked'`,
		swapID, userID, status)
	if err != nil {
		return fmt.Errorf("close hold: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("close hold: no locked hold for swap %s and user %s", swapID, userID)
	}
	return nil
}

// SaveSwap writes the fields that change during a swap's life.
func (t *swapTx) SaveSwap(ctx context.Context, s domain.Swap) error {
	_, err := t.tx.Exec(ctx, `
		UPDATE swaps
		SET status = $2, outcome_a = $3, outcome_b = $4, cancel_requested_by = $5,
		    accepted_at = $6, deadline_at = $7, closed_at = $8
		WHERE id = $1`,
		s.ID, s.Status, s.OutcomeA, s.OutcomeB, s.CancelRequestedBy,
		s.AcceptedAt, s.DeadlineAt, s.ClosedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "swaps_one_active_per_project" {
		return &domain.ConflictError{Message: "this project already has an active swap"}
	}
	if err != nil {
		return fmt.Errorf("save swap: %w", err)
	}
	return nil
}

func (t *swapTx) SetProjectStatus(ctx context.Context, projectID string, status domain.ProjectStatus) error {
	_, err := t.tx.Exec(ctx, `UPDATE projects SET status = $2 WHERE id = $1`, projectID, status)
	if err != nil {
		return fmt.Errorf("set project status: %w", err)
	}
	return nil
}

func (t *swapTx) RejectOtherProposals(ctx context.Context, projectID, acceptedSwapID string) error {
	_, err := t.tx.Exec(ctx, `
		UPDATE swaps SET status = 'rejected', closed_at = now()
		WHERE project_id = $1 AND id <> $2 AND status = 'proposed'`,
		projectID, acceptedSwapID)
	if err != nil {
		return fmt.Errorf("reject other proposals: %w", err)
	}
	return nil
}
