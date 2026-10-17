package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// SwapStore is everything the swap service needs from the database.
type SwapStore interface {
	GetProject(ctx context.Context, id string) (domain.Project, error)
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	GetCreatorStats(ctx context.Context, userID string) (domain.CreatorStats, error)
	CreateSwap(ctx context.Context, s domain.Swap) (domain.Swap, error)
	GetSwap(ctx context.Context, id string) (domain.Swap, error)
	ListSwaps(ctx context.Context, userID string, status domain.SwapStatus) ([]domain.Swap, error)
	InTx(ctx context.Context, fn func(tx domain.SwapTx) error) error
}

// SwapService holds the rules of the swap life cycle and of the money
// that moves with it.
type SwapService struct {
	store SwapStore
	now   func() time.Time
}

// NewSwapService builds the service.
func NewSwapService(store SwapStore) *SwapService {
	return &SwapService{store: store, now: time.Now}
}

// Calculate shows the terms of a possible swap without saving anything.
func (s *SwapService) Calculate(ctx context.Context, userID string, p domain.SwapProposal) (domain.Swap, error) {
	return s.build(ctx, userID, p)
}

// Propose saves a new swap proposal. No money moves yet.
func (s *SwapService) Propose(ctx context.Context, userID string, p domain.SwapProposal) (domain.Swap, error) {
	swap, err := s.build(ctx, userID, p)
	if err != nil {
		return domain.Swap{}, err
	}
	return s.store.CreateSwap(ctx, swap)
}

// build checks a proposal and calculates its terms.
func (s *SwapService) build(ctx context.Context, userID string, p domain.SwapProposal) (domain.Swap, error) {
	if err := p.Validate(); err != nil {
		return domain.Swap{}, err
	}
	if p.PartnerID == userID {
		return domain.Swap{}, &domain.ValidationError{Field: "partner_id", Message: "you cannot make a swap with yourself"}
	}

	project, err := s.store.GetProject(ctx, p.ProjectID)
	if err != nil {
		return domain.Swap{}, err
	}
	if project.CreatorID != userID {
		return domain.Swap{}, domain.ErrForbidden // only the creator offers swaps on a project
	}
	if project.Status != domain.ProjectOpen {
		return domain.Swap{}, &domain.ConflictError{Message: "only open projects can get a swap"}
	}

	creator, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return domain.Swap{}, err
	}
	partner, err := s.store.GetUserByID(ctx, p.PartnerID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Swap{}, &domain.ValidationError{Field: "partner_id", Message: "no user with this id"}
	}
	if err != nil {
		return domain.Swap{}, err
	}

	statsA, err := s.store.GetCreatorStats(ctx, creator.ID)
	if err != nil {
		return domain.Swap{}, err
	}
	statsB, err := s.store.GetCreatorStats(ctx, partner.ID)
	if err != nil {
		return domain.Swap{}, err
	}

	roleA, roleB := p.PartyARole, p.PartyBRole
	if roleA == "" {
		roleA = creator.Role
	}
	if roleB == "" {
		roleB = partner.Role
	}

	return domain.Swap{
		ProjectID: project.ID, ProjectTitle: project.Title,
		PartyAID: creator.ID, PartyAUsername: creator.Username, PartyARole: roleA,
		PartyBID: partner.ID, PartyBUsername: partner.Username, PartyBRole: roleB,
		Terms:  domain.CalculateSwapTerms(project.Budget, project.SuccessProbabilityBP, statsA, statsB),
		Status: domain.SwapProposed,
	}, nil
}

// Get returns one swap. Only its two parties may see it.
func (s *SwapService) Get(ctx context.Context, userID, id string) (domain.Swap, error) {
	if !domain.IsUUID(id) {
		return domain.Swap{}, domain.ErrNotFound
	}
	swap, err := s.store.GetSwap(ctx, id)
	if err != nil {
		return domain.Swap{}, err
	}
	if !swap.IsParty(userID) {
		return domain.Swap{}, domain.ErrNotFound // do not even reveal that it exists
	}
	return swap, nil
}

// List returns the user's swaps, optionally only those in one status.
func (s *SwapService) List(ctx context.Context, userID string, status domain.SwapStatus) ([]domain.Swap, error) {
	switch status {
	case "", domain.SwapProposed, domain.SwapActive, domain.SwapCompleted,
		domain.SwapFailed, domain.SwapCancelled, domain.SwapRejected:
	default:
		return nil, &domain.ValidationError{Field: "status", Message: "unknown status"}
	}
	return s.store.ListSwaps(ctx, userID, status)
}

// Accept is party B saying yes. In ONE transaction: both guarantees are
// frozen, the swap becomes active, the project goes in progress and the
// other proposals for the project are rejected.
func (s *SwapService) Accept(ctx context.Context, userID, id string) (domain.Swap, error) {
	return s.change(ctx, userID, id, func(tx domain.SwapTx, swap *domain.Swap) error {
		if swap.PartyBID != userID {
			return domain.ErrForbidden // only the invited partner can accept
		}
		if swap.Status != domain.SwapProposed {
			return &domain.ConflictError{Message: "only a proposed swap can be accepted"}
		}
		project, err := tx.LockProject(ctx, swap.ProjectID)
		if err != nil {
			return err
		}
		if project.Status != domain.ProjectOpen {
			return &domain.ConflictError{Message: "the project is no longer open"}
		}

		if err := tx.LockWallets(ctx, swap.PartyAID, swap.PartyBID); err != nil {
			return err
		}
		if err := freeze(ctx, tx, swap, swap.PartyAID, swap.PartyAUsername, swap.Terms.PenaltyA); err != nil {
			return err
		}
		if err := freeze(ctx, tx, swap, swap.PartyBID, swap.PartyBUsername, swap.Terms.PenaltyB); err != nil {
			return err
		}

		now := s.now()
		deadline := now.AddDate(0, 0, project.DurationDays)
		swap.Status = domain.SwapActive
		swap.AcceptedAt, swap.DeadlineAt = &now, &deadline
		if err := tx.SaveSwap(ctx, *swap); err != nil {
			return err
		}
		if err := tx.SetProjectStatus(ctx, swap.ProjectID, domain.ProjectInProgress); err != nil {
			return err
		}
		return tx.RejectOtherProposals(ctx, swap.ProjectID, swap.ID)
	})
}

// Reject is party B saying no. No money has moved, so nothing to return.
func (s *SwapService) Reject(ctx context.Context, userID, id string) (domain.Swap, error) {
	return s.change(ctx, userID, id, func(tx domain.SwapTx, swap *domain.Swap) error {
		if swap.PartyBID != userID {
			return domain.ErrForbidden
		}
		if swap.Status != domain.SwapProposed {
			return &domain.ConflictError{Message: "only a proposed swap can be rejected"}
		}
		now := s.now()
		swap.Status, swap.ClosedAt = domain.SwapRejected, &now
		return tx.SaveSwap(ctx, *swap)
	})
}

// Cancel works in two ways:
//   - a proposed swap is cancelled at once by either party (no money moved);
//   - an active swap needs BOTH parties: the first call only records the
//     request, the second call (by the other party) cancels the swap and
//     gives each party its full guarantee back, without a fee.
func (s *SwapService) Cancel(ctx context.Context, userID, id string) (domain.Swap, error) {
	return s.change(ctx, userID, id, func(tx domain.SwapTx, swap *domain.Swap) error {
		now := s.now()
		switch swap.Status {
		case domain.SwapProposed:
			swap.Status, swap.ClosedAt = domain.SwapCancelled, &now
			return tx.SaveSwap(ctx, *swap)

		case domain.SwapActive:
			if swap.CancelRequestedBy == nil {
				swap.CancelRequestedBy = &userID // first party asks
				return tx.SaveSwap(ctx, *swap)
			}
			if *swap.CancelRequestedBy == userID {
				return &domain.ConflictError{Message: "you already asked to cancel; waiting for the other party"}
			}
			// The second party agrees: everybody gets their money back.
			if err := tx.LockWallets(ctx, swap.PartyAID, swap.PartyBID); err != nil {
				return err
			}
			if err := release(ctx, tx, swap, swap.PartyAID, swap.Terms.PenaltyA, 0); err != nil {
				return err
			}
			if err := release(ctx, tx, swap, swap.PartyBID, swap.Terms.PenaltyB, 0); err != nil {
				return err
			}
			swap.Status, swap.ClosedAt = domain.SwapCancelled, &now
			if err := tx.SaveSwap(ctx, *swap); err != nil {
				return err
			}
			return tx.SetProjectStatus(ctx, swap.ProjectID, domain.ProjectOpen)

		default:
			return &domain.ConflictError{Message: "this swap is already closed"}
		}
	})
}

// ReportOutcome records what one party says about the result. When both
// parties have said the same thing, the swap closes and the money moves:
//   - both "success": each party gets its guarantee back, minus the 0.5% fee;
//   - both "failure": each guarantee (minus the fee) goes to the OTHER party.
//
// If they disagree, the swap stays active and either party may change its
// answer. (A real dispute process with an arbiter is outside this project.)
func (s *SwapService) ReportOutcome(ctx context.Context, userID, id string, outcome domain.Outcome) (domain.Swap, error) {
	return s.change(ctx, userID, id, func(tx domain.SwapTx, swap *domain.Swap) error {
		if swap.Status != domain.SwapActive {
			return &domain.ConflictError{Message: "only an active swap can get an outcome"}
		}
		if userID == swap.PartyAID {
			swap.OutcomeA = &outcome
		} else {
			swap.OutcomeB = &outcome
		}

		if swap.OutcomeA == nil || swap.OutcomeB == nil || *swap.OutcomeA != *swap.OutcomeB {
			return tx.SaveSwap(ctx, *swap) // wait for the other party, or for agreement
		}

		if err := tx.LockWallets(ctx, swap.PartyAID, swap.PartyBID); err != nil {
			return err
		}
		now := s.now()
		swap.ClosedAt = &now
		projectStatus := domain.ProjectCompleted

		if outcome == domain.OutcomeSuccess {
			swap.Status = domain.SwapCompleted
			if err := release(ctx, tx, swap, swap.PartyAID, swap.Terms.PenaltyA, domain.Fee(swap.Terms.PenaltyA)); err != nil {
				return err
			}
			if err := release(ctx, tx, swap, swap.PartyBID, swap.Terms.PenaltyB, domain.Fee(swap.Terms.PenaltyB)); err != nil {
				return err
			}
		} else {
			swap.Status = domain.SwapFailed
			projectStatus = domain.ProjectFailed
			if err := payPenalty(ctx, tx, swap, swap.PartyAID, swap.PartyBID, swap.Terms.PenaltyA); err != nil {
				return err
			}
			if err := payPenalty(ctx, tx, swap, swap.PartyBID, swap.PartyAID, swap.Terms.PenaltyB); err != nil {
				return err
			}
		}

		if err := tx.SaveSwap(ctx, *swap); err != nil {
			return err
		}
		return tx.SetProjectStatus(ctx, swap.ProjectID, projectStatus)
	})
}

// change runs one rule inside a transaction: lock the swap, check that the
// user is one of its parties, apply the rule, then read the saved result.
func (s *SwapService) change(ctx context.Context, userID, id string, rule func(tx domain.SwapTx, swap *domain.Swap) error) (domain.Swap, error) {
	if !domain.IsUUID(id) {
		return domain.Swap{}, domain.ErrNotFound
	}
	err := s.store.InTx(ctx, func(tx domain.SwapTx) error {
		swap, err := tx.LockSwap(ctx, id)
		if err != nil {
			return err
		}
		if !swap.IsParty(userID) {
			return domain.ErrNotFound
		}
		return rule(tx, &swap)
	})
	if err != nil {
		return domain.Swap{}, err
	}
	return s.store.GetSwap(ctx, id)
}

// ----- money movements (always inside the caller's transaction) -----

// freeze moves a party's guarantee from balance to frozen.
func freeze(ctx context.Context, tx domain.SwapTx, swap *domain.Swap, userID, username string, amount int64) error {
	err := tx.MoveMoney(ctx, domain.MoneyMove{
		UserID: userID, SwapID: swap.ID, Type: domain.TxEscrowLock, Amount: amount,
		BalanceDelta: -amount, FrozenDelta: amount, Description: "guarantee for: " + swap.ProjectTitle,
	})
	if errors.Is(err, domain.ErrInsufficientFunds) {
		return &domain.ConflictError{Message: fmt.Sprintf(
			"%s does not have enough money for the guarantee (needs %d kopecks)", username, amount)}
	}
	if err != nil {
		return err
	}
	return tx.CreateHold(ctx, swap.ID, userID, amount)
}

// release gives a guarantee back to its owner, minus the fee.
func release(ctx context.Context, tx domain.SwapTx, swap *domain.Swap, userID string, amount, fee int64) error {
	if err := chargeFee(ctx, tx, swap, userID, fee); err != nil {
		return err
	}
	back := amount - fee
	if err := tx.MoveMoney(ctx, domain.MoneyMove{
		UserID: userID, SwapID: swap.ID, Type: domain.TxEscrowRelease, Amount: back,
		BalanceDelta: back, FrozenDelta: -back, Description: "guarantee returned: " + swap.ProjectTitle,
	}); err != nil {
		return err
	}
	return tx.CloseHold(ctx, swap.ID, userID, domain.HoldReleased)
}

// payPenalty takes the payer's guarantee (minus the fee) and gives it to
// the receiver.
func payPenalty(ctx context.Context, tx domain.SwapTx, swap *domain.Swap, payerID, receiverID string, amount int64) error {
	fee := domain.Fee(amount)
	if err := chargeFee(ctx, tx, swap, payerID, fee); err != nil {
		return err
	}
	paid := amount - fee
	if err := tx.MoveMoney(ctx, domain.MoneyMove{
		UserID: payerID, SwapID: swap.ID, Type: domain.TxPenaltyPaid, Amount: paid,
		FrozenDelta: -paid, Description: "penalty paid: " + swap.ProjectTitle,
	}); err != nil {
		return err
	}
	if err := tx.MoveMoney(ctx, domain.MoneyMove{
		UserID: receiverID, SwapID: swap.ID, Type: domain.TxPenaltyReceived, Amount: paid,
		BalanceDelta: paid, Description: "penalty received: " + swap.ProjectTitle,
	}); err != nil {
		return err
	}
	return tx.CloseHold(ctx, swap.ID, payerID, domain.HoldForfeited)
}

// chargeFee takes the platform fee from the frozen guarantee.
func chargeFee(ctx context.Context, tx domain.SwapTx, swap *domain.Swap, userID string, fee int64) error {
	if fee == 0 {
		return nil
	}
	return tx.MoveMoney(ctx, domain.MoneyMove{
		UserID: userID, SwapID: swap.ID, Type: domain.TxFee, Amount: fee,
		FrozenDelta: -fee, Description: "platform fee 0.5%: " + swap.ProjectTitle,
	})
}
