package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// ProjectRepository reads and writes projects.
type ProjectRepository struct {
	pool *pgxpool.Pool
}

// NewProjectRepository creates the repository on top of the shared pool.
func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

// projectColumns and projectFrom are shared by every query that reads
// projects, so the column order always matches scanProject.
const projectColumns = `
	p.id, p.creator_id, u.username, p.title, p.description, p.category,
	p.budget, p.duration_days, p.complexity, p.team_size, p.kpis,
	p.success_probability_bp, p.status, p.created_at, p.updated_at`

const projectFrom = `
	FROM projects p
	JOIN users u ON u.id = p.creator_id`

// scanProject reads one row. extra receives any columns after the project
// columns (the list query adds the total count there).
func scanProject(row pgx.Row, extra ...any) (domain.Project, error) {
	var p domain.Project
	var kpis []byte
	dest := []any{&p.ID, &p.CreatorID, &p.CreatorUsername, &p.Title, &p.Description, &p.Category,
		&p.Budget, &p.DurationDays, &p.Complexity, &p.TeamSize, &kpis,
		&p.SuccessProbabilityBP, &p.Status, &p.CreatedAt, &p.UpdatedAt}
	err := row.Scan(append(dest, extra...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Project{}, fmt.Errorf("read project: %w", err)
	}
	if err := json.Unmarshal(kpis, &p.KPIs); err != nil {
		return domain.Project{}, fmt.Errorf("read project kpis: %w", err)
	}
	return p, nil
}

// CreateProject inserts a new project and returns it as stored.
func (r *ProjectRepository) CreateProject(ctx context.Context, creatorID string, in domain.ProjectInput, successBP int) (domain.Project, error) {
	kpis, err := json.Marshal(in.KPIs)
	if err != nil {
		return domain.Project{}, fmt.Errorf("encode kpis: %w", err)
	}

	var id string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO projects (creator_id, title, description, category, budget,
		                      duration_days, complexity, team_size, kpis, success_probability_bp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		creatorID, in.Title, in.Description, in.Category, in.Budget,
		in.DurationDays, in.Complexity, in.TeamSize, kpis, successBP,
	).Scan(&id)
	if err != nil {
		return domain.Project{}, fmt.Errorf("insert project: %w", err)
	}
	return r.GetProject(ctx, id)
}

// GetProject finds one project by ID.
func (r *ProjectRepository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	return scanProject(r.pool.QueryRow(ctx, `SELECT `+projectColumns+projectFrom+` WHERE p.id = $1`, id))
}

// ListProjects returns one page of projects and the total number that match.
func (r *ProjectRepository) ListProjects(ctx context.Context, f domain.ProjectFilter) ([]domain.Project, int, error) {
	// Build the WHERE part step by step. Every value goes in args and is
	// referenced as $1, $2, ... so user input never becomes part of the SQL.
	var where []string
	var args []any
	add := func(condition string, value any) {
		args = append(args, value)
		where = append(where, strings.ReplaceAll(condition, "?", "$"+strconv.Itoa(len(args))))
	}

	add("p.status = ?", f.Status)
	if f.Category != "" {
		add("p.category = ?", f.Category)
	}
	if f.CreatorID != "" {
		add("p.creator_id = ?", f.CreatorID)
	}
	if f.Query != "" {
		add("(p.title ILIKE ? OR p.description ILIKE ?)", "%"+escapeLike(f.Query)+"%")
	}

	args = append(args, f.Limit, f.Offset)
	// count(*) OVER () adds the total number of matching rows to every
	// row, so one query gives both the page and the total.
	query := `SELECT ` + projectColumns + `, count(*) OVER ()` + projectFrom +
		" WHERE " + strings.Join(where, " AND ") +
		fmt.Sprintf(" ORDER BY p.created_at DESC, p.id LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := []domain.Project{}
	total := 0
	for rows.Next() {
		p, err := scanProject(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	return projects, total, nil
}

// UpdateProjectDetails saves a new title, description and KPIs. The
// "status = 'open'" condition makes sure a project that left the open
// state a moment ago is not changed.
func (r *ProjectRepository) UpdateProjectDetails(ctx context.Context, p domain.Project) (domain.Project, error) {
	kpis, err := json.Marshal(p.KPIs)
	if err != nil {
		return domain.Project{}, fmt.Errorf("encode kpis: %w", err)
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE projects SET title = $2, description = $3, kpis = $4
		WHERE id = $1 AND status = 'open'`,
		p.ID, p.Title, p.Description, kpis)
	if err != nil {
		return domain.Project{}, fmt.Errorf("update project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Project{}, &domain.ConflictError{Message: "only open projects can be changed"}
	}
	return r.GetProject(ctx, p.ID)
}

// ChangeProjectStatus moves a project from one status to another. It
// returns false if the project was not in the "from" status any more.
func (r *ProjectRepository) ChangeProjectStatus(ctx context.Context, id string, from, to domain.ProjectStatus) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE projects SET status = $3
		WHERE id = $1 AND status = $2`, id, from, to)
	if err != nil {
		return false, fmt.Errorf("change project status: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// GetCreatorStats reads what the scoring formula needs about a user:
// the latest reputation snapshot and the account creation date.
func (r *ProjectRepository) GetCreatorStats(ctx context.Context, userID string) (domain.CreatorStats, error) {
	var s domain.CreatorStats
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(rs.score_bp, 0), COALESCE(rs.success_rate_bp, 0),
		       COALESCE(rs.completed_count, 0), COALESCE(rs.failed_count, 0), u.created_at
		FROM users u
		LEFT JOIN LATERAL (
			SELECT score_bp, success_rate_bp, completed_count, failed_count
			FROM reputation_snapshots
			WHERE user_id = u.id
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) rs ON true
		WHERE u.id = $1`, userID,
	).Scan(&s.ReputationBP, &s.SuccessRateBP, &s.Completed, &s.Failed, &s.AccountSince)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CreatorStats{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.CreatorStats{}, fmt.Errorf("read creator stats: %w", err)
	}
	return s, nil
}

// escapeLike makes %, _ and \ in the search text match literally,
// so searching for "100%" does not mean "100 followed by anything".
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
