package service

import (
	"context"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// WalletStore is what the wallet service needs from the database.
type WalletStore interface {
	GetWallet(ctx context.Context, userID string) (domain.Wallet, error)
	ListTransactions(ctx context.Context, userID string, limit, offset int) ([]domain.Transaction, int, error)
	Deposit(ctx context.Context, userID string, amount int64) (domain.Wallet, error)
}

// WalletService shows wallets and the ledger, and adds demo money.
type WalletService struct {
	store WalletStore
}

// NewWalletService builds the service.
func NewWalletService(store WalletStore) *WalletService {
	return &WalletService{store: store}
}

// Get returns the user's wallet.
func (s *WalletService) Get(ctx context.Context, userID string) (domain.Wallet, error) {
	return s.store.GetWallet(ctx, userID)
}

// Transactions returns one page of the user's ledger.
func (s *WalletService) Transactions(ctx context.Context, userID string, limit, offset int) ([]domain.Transaction, int, error) {
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return nil, 0, &domain.ValidationError{Field: "limit", Message: "must be 1-100"}
	}
	if offset < 0 {
		return nil, 0, &domain.ValidationError{Field: "offset", Message: "must be 0 or more"}
	}
	return s.store.ListTransactions(ctx, userID, limit, offset)
}

// Deposit adds demo money. The route only exists in development mode
// (see routes.go), so real deployments cannot create money from nothing.
func (s *WalletService) Deposit(ctx context.Context, userID string, amount int64) (domain.Wallet, error) {
	if err := domain.ValidateDeposit(amount); err != nil {
		return domain.Wallet{}, err
	}
	return s.store.Deposit(ctx, userID, amount)
}
