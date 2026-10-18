package service

import (
	"context"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// ReputationStore is everything the reputation service needs from the database.
type ReputationStore interface {
	GetUserByID(ctx context.Context, id string) (domain.User, error)
	GetSwap(ctx context.Context, id string) (domain.Swap, error)
	GetReputationFacts(ctx context.Context, userID string) (domain.ReputationFacts, error)
	SaveReputation(ctx context.Context, userID string, rep domain.Reputation) (domain.Reputation, error)
	GetLatestReputation(ctx context.Context, userID string) (domain.Reputation, error)
	ListReputationHistory(ctx context.Context, userID string, limit int) ([]domain.Reputation, error)
	CreateReview(ctx context.Context, rv domain.Review) (domain.Review, error)
	ListReviews(ctx context.Context, targetID string, limit int) ([]domain.Review, error)
}

// ReputationService recalculates reputation and handles reviews.
type ReputationService struct {
	store ReputationStore
	now   func() time.Time
}

// NewReputationService builds the service.
func NewReputationService(store ReputationStore) *ReputationService {
	return &ReputationService{store: store, now: time.Now}
}

// Recompute reads the user's facts, runs the formula and saves a new
// snapshot. reason says what caused it (swap_completed, review_received...).
func (s *ReputationService) Recompute(ctx context.Context, userID, reason string) (domain.Reputation, error) {
	facts, err := s.store.GetReputationFacts(ctx, userID)
	if err != nil {
		return domain.Reputation{}, err
	}
	rep := domain.CalculateReputation(facts, s.now(), reason)
	return s.store.SaveReputation(ctx, userID, rep)
}

// AddReview lets one party rate the other after the swap has finished.
func (s *ReputationService) AddReview(ctx context.Context, authorID, swapID string, in domain.ReviewInput) (domain.Review, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return domain.Review{}, err
	}
	if !domain.IsUUID(swapID) {
		return domain.Review{}, domain.ErrNotFound
	}
	swap, err := s.store.GetSwap(ctx, swapID)
	if err != nil {
		return domain.Review{}, err
	}
	if !swap.IsParty(authorID) {
		return domain.Review{}, domain.ErrNotFound // strangers cannot review
	}
	if swap.Status != domain.SwapCompleted && swap.Status != domain.SwapFailed {
		return domain.Review{}, &domain.ConflictError{Message: "you can review only a finished swap (completed or failed)"}
	}

	// The author rates the OTHER party.
	targetID, authorName := swap.PartyBID, swap.PartyAUsername
	if authorID == swap.PartyBID {
		targetID, authorName = swap.PartyAID, swap.PartyBUsername
	}

	review, err := s.store.CreateReview(ctx, domain.Review{
		SwapID: swap.ID, AuthorID: authorID, AuthorUsername: authorName, TargetID: targetID,
		Rating: in.Rating, Comment: in.Comment,
	})
	if err != nil {
		return domain.Review{}, err
	}
	if _, err := s.Recompute(ctx, targetID, "review_received"); err != nil {
		return domain.Review{}, err
	}
	return review, nil
}

// Profile is the public reputation page of one user.
type Profile struct {
	User    domain.User
	Current domain.Reputation
	History []domain.Reputation
	Reviews []domain.Review
}

// Profile collects the current reputation, its history and the reviews.
func (s *ReputationService) Profile(ctx context.Context, userID string) (Profile, error) {
	if !domain.IsUUID(userID) {
		return Profile{}, domain.ErrNotFound
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	current, err := s.store.GetLatestReputation(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	// Trust grows with account age, so it is computed now, not read from
	// the last snapshot (which may be days old).
	current.TrustWeightBP = domain.TrustWeight(int(s.now().Sub(user.CreatedAt).Hours()/24), current.CompletedCount)

	history, err := s.store.ListReputationHistory(ctx, userID, 50)
	if err != nil {
		return Profile{}, err
	}
	reviews, err := s.store.ListReviews(ctx, userID, 20)
	if err != nil {
		return Profile{}, err
	}
	return Profile{User: user, Current: current, History: history, Reviews: reviews}, nil
}
