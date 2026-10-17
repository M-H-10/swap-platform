package service

import (
	"context"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// ProjectStore is everything the project service needs from the database.
type ProjectStore interface {
	CreateProject(ctx context.Context, creatorID string, in domain.ProjectInput, successBP int) (domain.Project, error)
	GetProject(ctx context.Context, id string) (domain.Project, error)
	ListProjects(ctx context.Context, f domain.ProjectFilter) ([]domain.Project, int, error)
	UpdateProjectDetails(ctx context.Context, p domain.Project) (domain.Project, error)
	ChangeProjectStatus(ctx context.Context, id string, from, to domain.ProjectStatus) (bool, error)
	GetCreatorStats(ctx context.Context, userID string) (domain.CreatorStats, error)
}

// ProjectService holds the rules for projects.
type ProjectService struct {
	store ProjectStore
	now   func() time.Time // the clock; tests replace it with a fixed time
}

// NewProjectService builds the service.
func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{store: store, now: time.Now}
}

// Estimate calculates the score of a project without saving it, so the
// user can see the success probability while filling in the form.
func (s *ProjectService) Estimate(ctx context.Context, creatorID string, in domain.ProjectInput) (domain.ScoreResult, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return domain.ScoreResult{}, err
	}
	return s.score(ctx, creatorID, in)
}

// Create validates the project, scores it and saves it.
func (s *ProjectService) Create(ctx context.Context, creatorID string, in domain.ProjectInput) (domain.Project, domain.ScoreResult, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return domain.Project{}, domain.ScoreResult{}, err
	}
	result, err := s.score(ctx, creatorID, in)
	if err != nil {
		return domain.Project{}, domain.ScoreResult{}, err
	}
	p, err := s.store.CreateProject(ctx, creatorID, in, result.SuccessProbabilityBP)
	if err != nil {
		return domain.Project{}, domain.ScoreResult{}, err
	}
	return p, result, nil
}

// Get returns one project.
func (s *ProjectService) Get(ctx context.Context, id string) (domain.Project, error) {
	if !domain.IsUUID(id) {
		return domain.Project{}, domain.ErrNotFound
	}
	return s.store.GetProject(ctx, id)
}

// List returns one page of projects and the total count.
func (s *ProjectService) List(ctx context.Context, f domain.ProjectFilter) ([]domain.Project, int, error) {
	f.Normalize()
	if err := f.Validate(); err != nil {
		return nil, 0, err
	}
	return s.store.ListProjects(ctx, f)
}

// Update changes the title, description or KPIs. Only the creator may do
// it, and only while the project is still open.
func (s *ProjectService) Update(ctx context.Context, userID, id string, upd domain.ProjectUpdate) (domain.Project, error) {
	p, err := s.ownOpenProject(ctx, userID, id)
	if err != nil {
		return domain.Project{}, err
	}

	if upd.Title != nil {
		p.Title = *upd.Title
	}
	if upd.Description != nil {
		p.Description = *upd.Description
	}
	if upd.KPIs != nil {
		p.KPIs = *upd.KPIs
	}

	// Run the same checks as on creation, on the project as it would be.
	in := domain.ProjectInput{
		Title: p.Title, Description: p.Description, Category: p.Category, Budget: p.Budget,
		DurationDays: p.DurationDays, Complexity: p.Complexity, TeamSize: p.TeamSize, KPIs: p.KPIs,
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return domain.Project{}, err
	}
	p.Title, p.Description, p.KPIs = in.Title, in.Description, in.KPIs

	return s.store.UpdateProjectDetails(ctx, p)
}

// Cancel closes an open project. Only the creator may do it.
func (s *ProjectService) Cancel(ctx context.Context, userID, id string) (domain.Project, error) {
	if _, err := s.ownOpenProject(ctx, userID, id); err != nil {
		return domain.Project{}, err
	}
	changed, err := s.store.ChangeProjectStatus(ctx, id, domain.ProjectOpen, domain.ProjectCancelled)
	if err != nil {
		return domain.Project{}, err
	}
	if !changed {
		return domain.Project{}, &domain.ConflictError{Message: "only open projects can be cancelled"}
	}
	return s.store.GetProject(ctx, id)
}

// ownOpenProject loads a project and checks the two rules every change
// needs: the user is the creator, and the project is still open.
func (s *ProjectService) ownOpenProject(ctx context.Context, userID, id string) (domain.Project, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	if p.CreatorID != userID {
		return domain.Project{}, domain.ErrForbidden
	}
	if p.Status != domain.ProjectOpen {
		return domain.Project{}, &domain.ConflictError{Message: "only open projects can be changed"}
	}
	return p, nil
}

func (s *ProjectService) score(ctx context.Context, creatorID string, in domain.ProjectInput) (domain.ScoreResult, error) {
	stats, err := s.store.GetCreatorStats(ctx, creatorID)
	if err != nil {
		return domain.ScoreResult{}, err
	}
	return domain.CalculateScore(domain.NewScoreInput(in, stats, s.now())), nil
}
