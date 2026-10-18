package service

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// fakeSwapStore is an in-memory database. It is also its own transaction:
// InTx saves a copy of the money and the swaps before running fn, and puts
// the copy back if fn fails. That is exactly what ROLLBACK does.
type fakeSwapStore struct {
	projects map[string]domain.Project
	users    map[string]domain.User
	swaps    map[string]domain.Swap
	wallets  map[string]domain.Wallet
	holds    map[string]domain.HoldStatus // key: swapID + "/" + userID
	ledger   []domain.MoneyMove
}

const (
	aliID     = "00000000-0000-0000-0000-00000000000a"
	monaID    = "00000000-0000-0000-0000-00000000000b"
	projectID = "00000000-0000-0000-0000-0000000000f1"
	swapID    = "00000000-0000-0000-0000-0000000000e1"
)

func newFakeSwapStore() *fakeSwapStore {
	return &fakeSwapStore{
		projects: map[string]domain.Project{projectID: {
			ID: projectID, CreatorID: aliID, Title: "Online shop", Budget: 100_000_00,
			DurationDays: 30, SuccessProbabilityBP: 3740, Status: domain.ProjectOpen,
		}},
		users: map[string]domain.User{
			aliID:  {ID: aliID, Username: "ali_test", Role: domain.RoleDeveloper},
			monaID: {ID: monaID, Username: "mona_dev", Role: domain.RoleDesigner},
		},
		swaps:   map[string]domain.Swap{},
		wallets: map[string]domain.Wallet{aliID: {UserID: aliID}, monaID: {UserID: monaID}},
		holds:   map[string]domain.HoldStatus{},
	}
}

// ----- SwapStore -----

func (f *fakeSwapStore) GetProject(_ context.Context, id string) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakeSwapStore) GetUserByID(_ context.Context, id string) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeSwapStore) GetCreatorStats(_ context.Context, _ string) (domain.CreatorStats, error) {
	return domain.CreatorStats{ReputationBP: 5000}, nil // two new users
}

func (f *fakeSwapStore) CreateSwap(_ context.Context, s domain.Swap) (domain.Swap, error) {
	s.ID = swapID
	f.swaps[s.ID] = s
	return s, nil
}

func (f *fakeSwapStore) GetSwap(_ context.Context, id string) (domain.Swap, error) {
	s, ok := f.swaps[id]
	if !ok {
		return domain.Swap{}, domain.ErrNotFound
	}
	return s, nil
}

func (f *fakeSwapStore) ListSwaps(_ context.Context, _ string, _ domain.SwapStatus) ([]domain.Swap, error) {
	return nil, nil
}

func (f *fakeSwapStore) InTx(_ context.Context, fn func(tx domain.SwapTx) error) error {
	wallets, swaps, holds, projects := maps.Clone(f.wallets), maps.Clone(f.swaps), maps.Clone(f.holds), maps.Clone(f.projects)
	ledger := len(f.ledger)
	if err := fn(f); err != nil {
		f.wallets, f.swaps, f.holds, f.projects, f.ledger = wallets, swaps, holds, projects, f.ledger[:ledger]
		return err
	}
	return nil
}

// ----- domain.SwapTx -----

func (f *fakeSwapStore) LockSwap(ctx context.Context, id string) (domain.Swap, error) {
	return f.GetSwap(ctx, id)
}

func (f *fakeSwapStore) LockProject(ctx context.Context, id string) (domain.Project, error) {
	return f.GetProject(ctx, id)
}

func (f *fakeSwapStore) LockWallets(context.Context, ...string) error { return nil }

func (f *fakeSwapStore) MoveMoney(_ context.Context, m domain.MoneyMove) error {
	w := f.wallets[m.UserID]
	w.Balance += m.BalanceDelta
	w.FrozenBalance += m.FrozenDelta
	if w.Balance < 0 || w.FrozenBalance < 0 { // what the CHECK constraints do
		return domain.ErrInsufficientFunds
	}
	f.wallets[m.UserID] = w
	f.ledger = append(f.ledger, m)
	return nil
}

func (f *fakeSwapStore) CreateHold(_ context.Context, swapID, userID string, _ int64) error {
	f.holds[swapID+"/"+userID] = domain.HoldLocked
	return nil
}

func (f *fakeSwapStore) CloseHold(_ context.Context, swapID, userID string, status domain.HoldStatus) error {
	if f.holds[swapID+"/"+userID] != domain.HoldLocked {
		return errors.New("no locked hold")
	}
	f.holds[swapID+"/"+userID] = status
	return nil
}

func (f *fakeSwapStore) SaveSwap(_ context.Context, s domain.Swap) error {
	f.swaps[s.ID] = s
	return nil
}

func (f *fakeSwapStore) SetProjectStatus(_ context.Context, id string, status domain.ProjectStatus) error {
	p := f.projects[id]
	p.Status = status
	f.projects[id] = p
	return nil
}

func (f *fakeSwapStore) RejectOtherProposals(context.Context, string, string) error { return nil }

// ----- helpers -----

func newSwapTestService() (*SwapService, *fakeSwapStore) {
	store := newFakeSwapStore()
	svc := NewSwapService(store, nil)
	svc.now = func() time.Time { return time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC) }
	return svc, store
}

func (f *fakeSwapStore) give(userID string, amount int64) {
	w := f.wallets[userID]
	w.Balance += amount
	f.wallets[userID] = w
}

// proposeAndAccept makes Ali propose to Mona and Mona accept, with 10 000
// rubles in each wallet. Each guarantee is 5 000 rubles (500 000 kopecks).
func proposeAndAccept(t *testing.T) (*SwapService, *fakeSwapStore) {
	t.Helper()
	svc, store := newSwapTestService()
	store.give(aliID, 10_000_00)
	store.give(monaID, 10_000_00)
	ctx := context.Background()
	if _, err := svc.Propose(ctx, aliID, domain.SwapProposal{ProjectID: projectID, PartnerID: monaID}); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := svc.Accept(ctx, monaID, swapID); err != nil {
		t.Fatalf("accept: %v", err)
	}
	return svc, store
}

func wantWallet(t *testing.T, store *fakeSwapStore, userID string, balance, frozen int64) {
	t.Helper()
	w := store.wallets[userID]
	if w.Balance != balance || w.FrozenBalance != frozen {
		t.Fatalf("wallet %s = %d / %d frozen, want %d / %d", userID[len(userID)-1:], w.Balance, w.FrozenBalance, balance, frozen)
	}
}

// ----- tests -----

func TestProposeOnlyByCreator(t *testing.T) {
	svc, _ := newSwapTestService()
	_, err := svc.Propose(context.Background(), monaID, domain.SwapProposal{ProjectID: projectID, PartnerID: aliID})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestProposeUsesRolesFromProfiles(t *testing.T) {
	svc, _ := newSwapTestService()
	s, err := svc.Propose(context.Background(), aliID, domain.SwapProposal{ProjectID: projectID, PartnerID: monaID})
	if err != nil {
		t.Fatal(err)
	}
	if s.PartyARole != domain.RoleDeveloper || s.PartyBRole != domain.RoleDesigner || s.Status != domain.SwapProposed {
		t.Fatalf("unexpected swap: %+v", s)
	}
}

func TestAcceptFreezesBothGuarantees(t *testing.T) {
	_, store := proposeAndAccept(t)
	wantWallet(t, store, aliID, 5_000_00, 5_000_00)
	wantWallet(t, store, monaID, 5_000_00, 5_000_00)
	if store.swaps[swapID].Status != domain.SwapActive || store.projects[projectID].Status != domain.ProjectInProgress {
		t.Fatal("swap should be active and project in progress")
	}
}

func TestAcceptWithoutMoneyChangesNothing(t *testing.T) {
	svc, store := newSwapTestService()
	store.give(aliID, 10_000_00) // Mona has nothing
	ctx := context.Background()
	if _, err := svc.Propose(ctx, aliID, domain.SwapProposal{ProjectID: projectID, PartnerID: monaID}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Accept(ctx, monaID, swapID)
	var ce *domain.ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError, got %v", err)
	}
	// Ali's guarantee was frozen first, but the rollback must give it back.
	wantWallet(t, store, aliID, 10_000_00, 0)
	if store.swaps[swapID].Status != domain.SwapProposed {
		t.Fatal("the swap must still be proposed")
	}
}

func TestOnlyPartyBCanAccept(t *testing.T) {
	svc, store := newSwapTestService()
	store.give(aliID, 10_000_00)
	store.give(monaID, 10_000_00)
	ctx := context.Background()
	_, _ = svc.Propose(ctx, aliID, domain.SwapProposal{ProjectID: projectID, PartnerID: monaID})
	if _, err := svc.Accept(ctx, aliID, swapID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestBothSuccessReturnsGuaranteesMinusFee(t *testing.T) {
	svc, store := proposeAndAccept(t)
	ctx := context.Background()

	if _, err := svc.ReportOutcome(ctx, aliID, swapID, domain.OutcomeSuccess); err != nil {
		t.Fatal(err)
	}
	wantWallet(t, store, aliID, 5_000_00, 5_000_00) // nothing moves after one answer

	s, err := svc.ReportOutcome(ctx, monaID, swapID, domain.OutcomeSuccess)
	if err != nil || s.Status != domain.SwapCompleted {
		t.Fatalf("expected completed, got %s (%v)", s.Status, err)
	}
	// 5 000 rubles back, minus 0.5% = 25 rubles.
	wantWallet(t, store, aliID, 9_975_00, 0)
	wantWallet(t, store, monaID, 9_975_00, 0)
	if store.projects[projectID].Status != domain.ProjectCompleted {
		t.Fatal("project should be completed")
	}
}

func TestBothFailurePaysPenalties(t *testing.T) {
	svc, store := proposeAndAccept(t)
	ctx := context.Background()
	_, _ = svc.ReportOutcome(ctx, aliID, swapID, domain.OutcomeFailure)
	if _, err := svc.ReportOutcome(ctx, monaID, swapID, domain.OutcomeFailure); err != nil {
		t.Fatal(err)
	}
	// Each paid 4 975 to the other and received 4 975: equal penalties
	// cancel out, only the fee is lost.
	wantWallet(t, store, aliID, 9_975_00, 0)
	wantWallet(t, store, monaID, 9_975_00, 0)
	if store.holds[swapID+"/"+aliID] != domain.HoldForfeited {
		t.Fatal("Ali's hold should be forfeited")
	}
}

func TestDisagreementKeepsSwapActive(t *testing.T) {
	svc, store := proposeAndAccept(t)
	ctx := context.Background()
	_, _ = svc.ReportOutcome(ctx, aliID, swapID, domain.OutcomeSuccess)
	s, err := svc.ReportOutcome(ctx, monaID, swapID, domain.OutcomeFailure)
	if err != nil || s.Status != domain.SwapActive {
		t.Fatalf("expected active, got %s (%v)", s.Status, err)
	}
	wantWallet(t, store, aliID, 5_000_00, 5_000_00)
}

func TestActiveCancelNeedsBothParties(t *testing.T) {
	svc, store := proposeAndAccept(t)
	ctx := context.Background()

	s, err := svc.Cancel(ctx, aliID, swapID)
	if err != nil || s.Status != domain.SwapActive {
		t.Fatalf("first cancel should only record the request, got %s (%v)", s.Status, err)
	}
	var ce *domain.ConflictError
	if _, err := svc.Cancel(ctx, aliID, swapID); !errors.As(err, &ce) {
		t.Fatalf("asking twice should be a conflict, got %v", err)
	}

	s, err = svc.Cancel(ctx, monaID, swapID)
	if err != nil || s.Status != domain.SwapCancelled {
		t.Fatalf("second party should cancel, got %s (%v)", s.Status, err)
	}
	// Full refund, no fee, and the project is open again.
	wantWallet(t, store, aliID, 10_000_00, 0)
	wantWallet(t, store, monaID, 10_000_00, 0)
	if store.projects[projectID].Status != domain.ProjectOpen {
		t.Fatal("project should be open again")
	}
}

func TestStrangerCannotSeeSwap(t *testing.T) {
	svc, _ := proposeAndAccept(t)
	stranger := "00000000-0000-0000-0000-0000000000cc"
	if _, err := svc.Get(context.Background(), stranger, swapID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
