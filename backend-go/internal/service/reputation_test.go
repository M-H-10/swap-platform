package service

import (
	"context"
	"errors"
	"testing"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// ----- part 1: the swap service calls Recompute at the right moments -----

// recorder remembers every Recompute call instead of touching a database.
type recorder struct{ calls []string }

func (r *recorder) Recompute(_ context.Context, userID, reason string) (domain.Reputation, error) {
	r.calls = append(r.calls, userID+" "+reason)
	return domain.Reputation{}, nil
}

func TestClosedSwapUpdatesBothReputations(t *testing.T) {
	svc, _ := proposeAndAccept(t)
	rec := &recorder{}
	svc.reputation = rec
	ctx := context.Background()

	_, _ = svc.ReportOutcome(ctx, aliID, swapID, domain.OutcomeSuccess)
	if len(rec.calls) != 0 {
		t.Fatalf("one answer must not update reputation, got %v", rec.calls)
	}
	if _, err := svc.ReportOutcome(ctx, monaID, swapID, domain.OutcomeSuccess); err != nil {
		t.Fatal(err)
	}
	want := []string{aliID + " swap_completed", monaID + " swap_completed"}
	if len(rec.calls) != 2 || rec.calls[0] != want[0] || rec.calls[1] != want[1] {
		t.Fatalf("calls = %v, want %v", rec.calls, want)
	}
}

func TestMutualCancelUpdatesReputation(t *testing.T) {
	svc, _ := proposeAndAccept(t)
	rec := &recorder{}
	svc.reputation = rec
	ctx := context.Background()

	_, _ = svc.Cancel(ctx, aliID, swapID) // only a request
	if _, err := svc.Cancel(ctx, monaID, swapID); err != nil {
		t.Fatal(err)
	}
	if len(rec.calls) != 2 || rec.calls[0] != aliID+" swap_cancelled" {
		t.Fatalf("calls = %v", rec.calls)
	}
}

// ----- part 2: reviews -----

// fakeReputationStore has one swap and remembers reviews and snapshots.
type fakeReputationStore struct {
	swap      domain.Swap
	reviews   []domain.Review
	snapshots map[string][]domain.Reputation
}

func newFakeReputationStore(status domain.SwapStatus) *fakeReputationStore {
	return &fakeReputationStore{
		swap:      domain.Swap{ID: swapID, PartyAID: aliID, PartyBID: monaID, Status: status},
		snapshots: map[string][]domain.Reputation{},
	}
}

func (f *fakeReputationStore) GetUserByID(_ context.Context, id string) (domain.User, error) {
	return domain.User{ID: id}, nil
}

func (f *fakeReputationStore) GetSwap(_ context.Context, id string) (domain.Swap, error) {
	if id != f.swap.ID {
		return domain.Swap{}, domain.ErrNotFound
	}
	return f.swap, nil
}

func (f *fakeReputationStore) GetReputationFacts(_ context.Context, userID string) (domain.ReputationFacts, error) {
	facts := domain.ReputationFacts{Completed: 1}
	for _, rv := range f.reviews {
		if rv.TargetID == userID {
			facts.Reviews = append(facts.Reviews, domain.ReviewFact{Rating: rv.Rating})
		}
	}
	return facts, nil
}

func (f *fakeReputationStore) SaveReputation(_ context.Context, userID string, rep domain.Reputation) (domain.Reputation, error) {
	f.snapshots[userID] = append(f.snapshots[userID], rep)
	return rep, nil
}

func (f *fakeReputationStore) GetLatestReputation(context.Context, string) (domain.Reputation, error) {
	return domain.Reputation{}, nil
}

func (f *fakeReputationStore) ListReputationHistory(context.Context, string, int) ([]domain.Reputation, error) {
	return nil, nil
}

func (f *fakeReputationStore) CreateReview(_ context.Context, rv domain.Review) (domain.Review, error) {
	for _, old := range f.reviews {
		if old.SwapID == rv.SwapID && old.AuthorID == rv.AuthorID {
			return domain.Review{}, &domain.ConflictError{Message: "you already reviewed this swap"}
		}
	}
	f.reviews = append(f.reviews, rv)
	return rv, nil
}

func (f *fakeReputationStore) ListReviews(context.Context, string, int) ([]domain.Review, error) {
	return nil, nil
}

func TestReviewGoesToTheOtherParty(t *testing.T) {
	store := newFakeReputationStore(domain.SwapCompleted)
	svc := NewReputationService(store)

	rv, err := svc.AddReview(context.Background(), monaID, swapID, domain.ReviewInput{Rating: 5, Comment: "  great  "})
	if err != nil {
		t.Fatal(err)
	}
	if rv.TargetID != aliID || rv.Comment != "great" {
		t.Fatalf("unexpected review: %+v", rv)
	}
	snaps := store.snapshots[aliID]
	if len(snaps) != 1 || snaps[0].Reason != "review_received" {
		t.Fatalf("Ali's reputation should be recalculated once, got %+v", snaps)
	}
}

func TestReviewRules(t *testing.T) {
	ctx := context.Background()
	var conflict *domain.ConflictError

	// the swap is still active
	svc := NewReputationService(newFakeReputationStore(domain.SwapActive))
	if _, err := svc.AddReview(ctx, aliID, swapID, domain.ReviewInput{Rating: 4}); !errors.As(err, &conflict) {
		t.Fatalf("active swap: expected conflict, got %v", err)
	}

	// a stranger
	svc = NewReputationService(newFakeReputationStore(domain.SwapCompleted))
	stranger := "00000000-0000-0000-0000-0000000000cc"
	if _, err := svc.AddReview(ctx, stranger, swapID, domain.ReviewInput{Rating: 4}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("stranger: expected not found, got %v", err)
	}

	// the second review of the same swap
	if _, err := svc.AddReview(ctx, aliID, swapID, domain.ReviewInput{Rating: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddReview(ctx, aliID, swapID, domain.ReviewInput{Rating: 5}); !errors.As(err, &conflict) {
		t.Fatalf("second review: expected conflict, got %v", err)
	}
}
