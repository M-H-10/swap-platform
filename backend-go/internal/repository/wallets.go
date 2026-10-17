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

// WalletRepository reads wallets and the money ledger.
type WalletRepository struct {
	pool *pgxpool.Pool
}

// NewWalletRepository creates the repository on top of the shared pool.
func NewWalletRepository(pool *pgxpool.Pool) *WalletRepository {
	return &WalletRepository{pool: pool}
}

// GetWallet returns the wallet of one user.
func (r *WalletRepository) GetWallet(ctx context.Context, userID string) (domain.Wallet, error) {
	var w domain.Wallet
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, balance, frozen_balance, updated_at
		FROM wallets WHERE user_id = $1`, userID,
	).Scan(&w.UserID, &w.Balance, &w.FrozenBalance, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Wallet{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("read wallet: %w", err)
	}
	return w, nil
}

// ListTransactions returns one page of a user's ledger, newest first.
func (r *WalletRepository) ListTransactions(ctx context.Context, userID string, limit, offset int) ([]domain.Transaction, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, swap_id, type, amount, balance_after, frozen_after,
		       description, created_at, count(*) OVER ()
		FROM transactions
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	list := []domain.Transaction{}
	total := 0
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.SwapID, &t.Type, &t.Amount, &t.BalanceAfter,
			&t.FrozenAfter, &t.Description, &t.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("read transaction: %w", err)
		}
		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	return list, total, nil
}

// Deposit adds demo money to a wallet and writes the ledger line, in one
// transaction.
func (r *WalletRepository) Deposit(ctx context.Context, userID string, amount int64) (domain.Wallet, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Wallet{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = moveMoney(ctx, tx, domain.MoneyMove{
		UserID: userID, Type: domain.TxDeposit, Amount: amount,
		BalanceDelta: amount, Description: "demo deposit",
	})
	if err != nil {
		return domain.Wallet{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Wallet{}, fmt.Errorf("commit: %w", err)
	}
	return r.GetWallet(ctx, userID)
}

// moveMoney is the ONLY function that changes a wallet. It updates the
// two numbers and writes the matching ledger line with the new values,
// inside the caller's transaction. The CHECK (balance >= 0) constraints
// of the wallets table make it impossible to spend money that is not there.
func moveMoney(ctx context.Context, tx pgx.Tx, m domain.MoneyMove) error {
	var balance, frozen int64
	err := tx.QueryRow(ctx, `
		UPDATE wallets
		SET balance = balance + $2, frozen_balance = frozen_balance + $3
		WHERE user_id = $1
		RETURNING balance, frozen_balance`,
		m.UserID, m.BalanceDelta, m.FrozenDelta,
	).Scan(&balance, &frozen)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" { // 23514 = check_violation
		return domain.ErrInsufficientFunds
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update wallet: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (user_id, swap_id, type, amount, balance_after, frozen_after, description)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7)`,
		m.UserID, m.SwapID, m.Type, m.Amount, balance, frozen, m.Description)
	if err != nil {
		return fmt.Errorf("write ledger: %w", err)
	}
	return nil
}
