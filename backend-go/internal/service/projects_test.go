package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// fakeProjectStore keeps projects in memory instead of PostgreSQL.
type fakeProjectStore struct {
	projects map[string]domain.Project
	stats    map[string]domain.CreatorStats
	nextID   int
}

func newFakeProjectStore() *fakeProjectStore {
	return &fakeProjectStore{projects: map[string]domain.Project{}, stats: map[string]domain.CreatorStats{}}
}

func (f *fakeProjectStore) CreateProject(_ context.Context, creatorID string, in domain.ProjectInput, successBP int) (domain.Project, error) {
	f.nextID++
	p := domain.Project{
		ID: fmt.Sprintf("00000000-0000-0000-0000-%012d", f.nextID), CreatorID: creatorID,
		Title: in.Title, Description: in.Description, Category: in.Category, Budget: in.Budget,
		DurationDays: in.DurationDays, Complexity: in.Complexity, TeamSize: in.TeamSize, KPIs: in.KPIs,
		SuccessProbabilityBP: successBP, Status: domain.ProjectOpen,
	}
	f.projects[p.ID] = p
	return p, nil
}

func (f *fakeProjectStore) GetProject(_ context.Context, id string) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, nil
}

func (f *fakeProjectStore) ListProjects(_ context.Context, fl domain.ProjectFilter) ([]domain.Project, int, error) {
	out := []domain.Project{}
	for _, p := range f.projects {
		if p.Status == fl.Status {
			out = append(out, p)
		}
	}
	return out, len(out), nil
}

func (f *fakeProjectStore) UpdateProjectDetails(_ context.Context, p domain.Project) (domain.Project, error) {
	f.projects[p.ID] = p
	return p, nil
}

func (f *fakeProjectStore) ChangeProjectStatus(_ context.Context, id string, from, to domain.ProjectStatus) (bool, error) {
	p, ok := f.projects[id]
	if !ok || p.Status != from {
		return false, nil
	}
	p.Status = to
	f.projects[id] = p
	return true, nil
}

func (f *fakeProjectStore) GetCreatorStats(_ context.Context, userID string) (domain.CreatorStats, error) {
	s, ok := f.stats[userID]
	if !ok {
		return domain.CreatorStats{}, domain.ErrNotFound
	}
	return s, nil
}

var testNow = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

// newProjectTestService returns a service where "owner" is a brand new
// account with the starting reputation, and "other" is a second user.
func newProjectTestService() (*ProjectService, *fakeProjectStore) {
	store := newFakeProjectStore()
	store.stats["owner"] = domain.CreatorStats{ReputationBP: 5000, AccountSince: testNow}
	store.stats["other"] = domain.CreatorStats{ReputationBP: 5000, AccountSince: testNow}
	svc := NewProjectService(store, nil)
	svc.now = func() time.Time { return testNow }
	return svc, store
}

func shopInput() domain.ProjectInput {
	return domain.ProjectInput{
		Title: "  Online shop ", Description: "A small online shop for handmade goods.",
		Category: "WEB", Budget: 100_000_00, DurationDays: 30, Complexity: 5, TeamSize: 2,
	}
}

func TestCreateProjectStoresScore(t *testing.T) {
	svc, _ := newProjectTestService()
	p, score, err := svc.Create(context.Background(), "owner", shopInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.Title != "Online shop" || p.Category != domain.CategoryWeb {
		t.Fatalf("input was not normalized: %q %q", p.Title, p.Category)
	}
	// Same numbers as the "new user, average project" case in scoring_test.go.
	if score.SuccessProbabilityBP != 3740 || p.SuccessProbabilityBP != 3740 {
		t.Fatalf("probability = %d (stored %d), want 3740", score.SuccessProbabilityBP, p.SuccessProbabilityBP)
	}
}

func TestCreateProjectRejectsBadInput(t *testing.T) {
	svc, store := newProjectTestService()
	in := shopInput()
	in.Budget = 0
	_, _, err := svc.Create(context.Background(), "owner", in)
	var ve *domain.ValidationError
	if !errors.As(err, &ve) || ve.Field != "budget" {
		t.Fatalf("expected budget validation error, got %v", err)
	}
	if len(store.projects) != 0 {
		t.Fatal("an invalid project was saved")
	}
}

func TestOnlyOwnerCanUpdate(t *testing.T) {
	svc, _ := newProjectTestService()
	ctx := context.Background()
	p, _, _ := svc.Create(ctx, "owner", shopInput())

	title := "Somebody else's title"
	if _, err := svc.Update(ctx, "other", p.ID, domain.ProjectUpdate{Title: &title}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	title = "Online shop v2"
	updated, err := svc.Update(ctx, "owner", p.ID, domain.ProjectUpdate{Title: &title})
	if err != nil || updated.Title != "Online shop v2" {
		t.Fatalf("owner update failed: %v (%q)", err, updated.Title)
	}
}

func TestCancelledProjectCannotChange(t *testing.T) {
	svc, _ := newProjectTestService()
	ctx := context.Background()
	p, _, _ := svc.Create(ctx, "owner", shopInput())

	cancelled, err := svc.Cancel(ctx, "owner", p.ID)
	if err != nil || cancelled.Status != domain.ProjectCancelled {
		t.Fatalf("cancel failed: %v (%s)", err, cancelled.Status)
	}

	var ce *domain.ConflictError
	if _, err := svc.Cancel(ctx, "owner", p.ID); !errors.As(err, &ce) {
		t.Fatalf("second cancel should be a conflict, got %v", err)
	}
	title := "Too late"
	if _, err := svc.Update(ctx, "owner", p.ID, domain.ProjectUpdate{Title: &title}); !errors.As(err, &ce) {
		t.Fatalf("update after cancel should be a conflict, got %v", err)
	}
}

func TestGetWithBadIDIsNotFound(t *testing.T) {
	svc, _ := newProjectTestService()
	if _, err := svc.Get(context.Background(), "abc"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
