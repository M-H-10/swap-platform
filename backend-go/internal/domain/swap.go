package domain

import (
	"context"
	"strings"
	"time"
)

// SwapStatus is where a swap is in its life.
type SwapStatus string

const (
	SwapProposed  SwapStatus = "proposed"  // waiting for party B
	SwapActive    SwapStatus = "active"    // accepted, money is frozen
	SwapCompleted SwapStatus = "completed" // both said "success"
	SwapFailed    SwapStatus = "failed"    // both said "failure"
	SwapCancelled SwapStatus = "cancelled" // withdrawn, or cancelled by both
	SwapRejected  SwapStatus = "rejected"  // party B said no
)

// Outcome is what one party says about how the project ended.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
)

// HoldStatus is what happened to the money one party locked.
type HoldStatus string

const (
	HoldLocked    HoldStatus = "locked"
	HoldReleased  HoldStatus = "released"  // given back to its owner
	HoldForfeited HoldStatus = "forfeited" // paid to the other party
)

// The money rules, in basis points (10000 bp = 100%).
const (
	EscrowRateBP = 1000 // the guarantee is 10% of the project budget
	FeeRateBP    = 50   // the platform keeps 0.5% of each guarantee when a swap closes
	minShareBP   = 500  // nobody gets less than 5% ...
	maxShareBP   = 9500 // ... or more than 95%
)

// RiskProfile shows one party what it can win or lose (kopecks).
type RiskProfile struct {
	MaxGain       int64 `json:"max_gain"`       // its share of the budget on success
	MaxLoss       int64 `json:"max_loss"`       // its penalty on failure
	ExpectedValue int64 `json:"expected_value"` // gain x P(success) - loss x P(failure)
}

// SwapTerms are the numbers of a swap contract.
type SwapTerms struct {
	ShareABP             int   // party A's share of the result
	ShareBBP             int   // party B's share (ShareABP + ShareBBP = 10000)
	EscrowAmount         int64 // total guarantee = PenaltyA + PenaltyB
	PenaltyA             int64 // what A locks, and loses on failure
	PenaltyB             int64 // what B locks, and loses on failure
	SuccessProbabilityBP int   // from the project's score
	RiskA                RiskProfile
	RiskB                RiskProfile
}

// Swap is a contract between the project creator (party A) and a partner
// (party B) that shares the risk of one project.
type Swap struct {
	ID                string
	ProjectID         string
	ProjectTitle      string
	PartyAID          string
	PartyAUsername    string
	PartyBID          string
	PartyBUsername    string
	PartyARole        Role
	PartyBRole        Role
	Terms             SwapTerms
	Status            SwapStatus
	OutcomeA          *Outcome
	OutcomeB          *Outcome
	CancelRequestedBy *string
	AcceptedAt        *time.Time
	DeadlineAt        *time.Time
	ClosedAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsParty reports whether the user is A or B in this swap.
func (s Swap) IsParty(userID string) bool {
	return userID == s.PartyAID || userID == s.PartyBID
}

// SwapProposal is what the project creator sends to start a swap.
type SwapProposal struct {
	ProjectID  string
	PartnerID  string
	PartyARole Role // empty = the creator's own role
	PartyBRole Role // empty = the partner's own role
}

// Validate checks the IDs and the optional roles.
func (p SwapProposal) Validate() error {
	if !IsUUID(p.ProjectID) {
		return &ValidationError{"project_id", "must be a project id"}
	}
	if !IsUUID(p.PartnerID) {
		return &ValidationError{"partner_id", "must be a user id"}
	}
	for field, r := range map[string]Role{"party_a_role": p.PartyARole, "party_b_role": p.PartyBRole} {
		switch r {
		case "", RoleDeveloper, RoleDesigner, RoleManager, RoleOther:
		default:
			return &ValidationError{field, "must be developer, designer, manager or other"}
		}
	}
	return nil
}

// ParseOutcome turns "success" / "failure" into an Outcome.
func ParseOutcome(s string) (Outcome, error) {
	switch o := Outcome(strings.ToLower(strings.TrimSpace(s))); o {
	case OutcomeSuccess, OutcomeFailure:
		return o, nil
	}
	return "", &ValidationError{"outcome", "must be success or failure"}
}

// CalculateSwapTerms decides how the risk is shared.
//
// Shares: 70% comes from the two reputations, 30% from the two success
// rates (unknown history counts as average). The more trusted party gets
// the larger share of the result.
//
// Guarantee: 10% of the budget. Each party locks a penalty, and the party
// with the SMALLER share locks the LARGER part: it gains less on success,
// so it is the one that must prove it is serious.
func CalculateSwapTerms(budget int64, successProbabilityBP int, a, b CreatorStats) SwapTerms {
	repShareA := shareOf(a.ReputationBP, b.ReputationBP)
	rateShareA := shareOf(knownSuccessRate(a), knownSuccessRate(b))
	shareA := clamp(repShareA*7/10+rateShareA*3/10, minShareBP, maxShareBP)
	shareB := 10000 - shareA

	escrow := budget * EscrowRateBP / 10000
	penaltyA := escrow * int64(shareB) / 10000
	penaltyB := escrow - penaltyA // so the two always add up exactly

	return SwapTerms{
		ShareABP:             shareA,
		ShareBBP:             shareB,
		EscrowAmount:         escrow,
		PenaltyA:             penaltyA,
		PenaltyB:             penaltyB,
		SuccessProbabilityBP: successProbabilityBP,
		RiskA:                riskProfile(budget, shareA, penaltyA, successProbabilityBP),
		RiskB:                riskProfile(budget, shareB, penaltyB, successProbabilityBP),
	}
}

// Fee is the platform commission on a guarantee of this size.
func Fee(amount int64) int64 {
	return amount * FeeRateBP / 10000
}

// shareOf returns x's part of x+y in basis points (50/50 when both are 0).
func shareOf(x, y int) int {
	if x+y <= 0 {
		return 5000
	}
	return x * 10000 / (x + y)
}

// knownSuccessRate treats "no finished projects yet" as an average 50%.
func knownSuccessRate(c CreatorStats) int {
	if c.Completed+c.Failed == 0 {
		return 5000
	}
	return clamp(c.SuccessRateBP, 0, 10000)
}

func riskProfile(budget int64, shareBP int, penalty int64, probabilityBP int) RiskProfile {
	gain := budget * int64(shareBP) / 10000
	p := int64(probabilityBP)
	return RiskProfile{
		MaxGain:       gain,
		MaxLoss:       penalty,
		ExpectedValue: (gain*p - penalty*(10000-p)) / 10000,
	}
}

// SwapTx is everything the swap rules need inside ONE database
// transaction. The repository implements it with PostgreSQL; the tests
// implement it in memory. Either all the calls made inside a transaction
// are saved, or none of them.
type SwapTx interface {
	// LockSwap and LockProject read a row and lock it until the end of the
	// transaction, so two requests cannot change the same swap at once.
	LockSwap(ctx context.Context, id string) (Swap, error)
	LockProject(ctx context.Context, id string) (Project, error)
	// LockWallets locks the wallets of the given users.
	LockWallets(ctx context.Context, userIDs ...string) error
	// MoveMoney changes one wallet and writes the ledger line. It returns
	// ErrInsufficientFunds if the balance or frozen balance would go below 0.
	MoveMoney(ctx context.Context, m MoneyMove) error
	CreateHold(ctx context.Context, swapID, userID string, amount int64) error
	CloseHold(ctx context.Context, swapID, userID string, status HoldStatus) error
	SaveSwap(ctx context.Context, s Swap) error
	SetProjectStatus(ctx context.Context, projectID string, status ProjectStatus) error
	// RejectOtherProposals closes the other open proposals of a project
	// once one swap has been accepted.
	RejectOtherProposals(ctx context.Context, projectID, acceptedSwapID string) error
}
