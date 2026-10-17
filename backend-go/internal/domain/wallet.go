package domain

import "time"

// Wallet holds a user's money in kopecks.
//
//	Balance       = money the user can spend
//	FrozenBalance = money locked as a guarantee in active swaps
type Wallet struct {
	UserID        string
	Balance       int64
	FrozenBalance int64
	UpdatedAt     time.Time
}

// TransactionType says why money moved. The values match the CHECK
// constraint of the transactions table.
type TransactionType string

const (
	TxDeposit         TransactionType = "deposit"          // money added (demo only)
	TxEscrowLock      TransactionType = "escrow_lock"      // balance -> frozen
	TxEscrowRelease   TransactionType = "escrow_release"   // frozen -> balance
	TxPenaltyPaid     TransactionType = "penalty_paid"     // frozen -> the other party
	TxPenaltyReceived TransactionType = "penalty_received" // from the other party -> balance
	TxFee             TransactionType = "fee"              // platform commission
)

// Transaction is one line of the ledger. Lines are never changed or deleted.
type Transaction struct {
	ID           int64
	UserID       string
	SwapID       *string // nil for deposits
	Type         TransactionType
	Amount       int64 // always positive
	BalanceAfter int64
	FrozenAfter  int64
	Description  string
	CreatedAt    time.Time
}

// MoneyMove is one change to one wallet, written to the ledger together
// with the wallet update. BalanceDelta and FrozenDelta can be negative.
type MoneyMove struct {
	UserID       string
	SwapID       string // "" for deposits
	Type         TransactionType
	Amount       int64
	BalanceDelta int64
	FrozenDelta  int64
	Description  string
}

// MaxDeposit is the largest single demo deposit: 1 000 000 rubles.
const MaxDeposit = 1_000_000_00

// ValidateDeposit checks a demo deposit amount (kopecks).
func ValidateDeposit(amount int64) error {
	if amount < 1 || amount > MaxDeposit {
		return &ValidationError{"amount", "must be between 1 and 100000000 kopecks (1 million rubles)"}
	}
	return nil
}
