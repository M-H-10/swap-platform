package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// ProjectHandler exposes the project routes.
type ProjectHandler struct {
	projects *service.ProjectService
}

// NewProjectHandler creates the handler.
func NewProjectHandler(projects *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// ----- request and response shapes -----

// projectRequest is the body of POST /api/projects and /estimate.
// budget is in kopecks: 150000 rubles = 15000000.
type projectRequest struct {
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Category     string       `json:"category"`
	Budget       int64        `json:"budget"`
	DurationDays int          `json:"duration_days"`
	Complexity   int          `json:"complexity"`
	TeamSize     int          `json:"team_size"`
	KPIs         []domain.KPI `json:"kpis"`
}

func (r projectRequest) toInput() domain.ProjectInput {
	return domain.ProjectInput{
		Title: r.Title, Description: r.Description, Category: domain.Category(r.Category),
		Budget: r.Budget, DurationDays: r.DurationDays, Complexity: r.Complexity,
		TeamSize: r.TeamSize, KPIs: r.KPIs,
	}
}

// projectUpdateRequest is the body of PATCH /api/projects/:id. A field
// that is missing from the JSON stays nil and is not changed.
type projectUpdateRequest struct {
	Title       *string       `json:"title"`
	Description *string       `json:"description"`
	KPIs        *[]domain.KPI `json:"kpis"`
}

type creatorResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type projectResponse struct {
	ID                   string          `json:"id"`
	Creator              creatorResponse `json:"creator"`
	Title                string          `json:"title"`
	Description          string          `json:"description"`
	Category             string          `json:"category"`
	Budget               int64           `json:"budget"`
	DurationDays         int             `json:"duration_days"`
	Complexity           int             `json:"complexity"`
	TeamSize             int             `json:"team_size"`
	KPIs                 []domain.KPI    `json:"kpis"`
	SuccessProbabilityBP int             `json:"success_probability_bp"`
	Status               string          `json:"status"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type scoreFactorResponse struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
}

type scoreResponse struct {
	ScoreBP              int                   `json:"score_bp"`
	SuccessProbabilityBP int                   `json:"success_probability_bp"`
	Factors              []scoreFactorResponse `json:"factors"`
}

type projectListResponse struct {
	Items  []projectResponse `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

func toProjectResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID: p.ID, Creator: creatorResponse{ID: p.CreatorID, Username: p.CreatorUsername},
		Title: p.Title, Description: p.Description, Category: string(p.Category),
		Budget: p.Budget, DurationDays: p.DurationDays, Complexity: p.Complexity,
		TeamSize: p.TeamSize, KPIs: p.KPIs, SuccessProbabilityBP: p.SuccessProbabilityBP,
		Status: string(p.Status), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toScoreResponse(r domain.ScoreResult) scoreResponse {
	factors := make([]scoreFactorResponse, len(r.Factors))
	for i, f := range r.Factors {
		factors[i] = scoreFactorResponse{Name: f.Name, Points: f.Points}
	}
	return scoreResponse{ScoreBP: r.ScoreBP, SuccessProbabilityBP: r.SuccessProbabilityBP, Factors: factors}
}

// ----- handlers -----

// Create handles POST /api/projects.
func (h *ProjectHandler) Create(c fiber.Ctx) error {
	var req projectRequest
	if err := readJSON(c, &req); err != nil {
		return err
	}
	p, score, err := h.projects.Create(c.Context(), middleware.UserID(c), req.toInput())
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"project": toProjectResponse(p),
		"score":   toScoreResponse(score),
	})
}

// Estimate handles POST /api/projects/estimate: the score without saving.
func (h *ProjectHandler) Estimate(c fiber.Ctx) error {
	var req projectRequest
	if err := readJSON(c, &req); err != nil {
		return err
	}
	score, err := h.projects.Estimate(c.Context(), middleware.UserID(c), req.toInput())
	if err != nil {
		return err
	}
	return c.JSON(toScoreResponse(score))
}

// List handles GET /api/projects?status=&category=&q=&creator_id=&limit=&offset=
func (h *ProjectHandler) List(c fiber.Ctx) error {
	limit, err := queryInt(c, "limit")
	if err != nil {
		return err
	}
	offset, err := queryInt(c, "offset")
	if err != nil {
		return err
	}
	f := domain.ProjectFilter{
		Status:    domain.ProjectStatus(c.Query("status")),
		Category:  domain.Category(c.Query("category")),
		Query:     c.Query("q"),
		CreatorID: c.Query("creator_id"),
		Limit:     limit,
		Offset:    offset,
	}

	projects, total, err := h.projects.List(c.Context(), f)
	if err != nil {
		return err
	}
	items := make([]projectResponse, len(projects))
	for i, p := range projects {
		items[i] = toProjectResponse(p)
	}
	f.Normalize() // to report the limit that was really used
	return c.JSON(projectListResponse{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset})
}

// Get handles GET /api/projects/:id.
func (h *ProjectHandler) Get(c fiber.Ctx) error {
	p, err := h.projects.Get(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(toProjectResponse(p))
}

// Update handles PATCH /api/projects/:id.
func (h *ProjectHandler) Update(c fiber.Ctx) error {
	var req projectUpdateRequest
	if err := readJSON(c, &req); err != nil {
		return err
	}
	p, err := h.projects.Update(c.Context(), middleware.UserID(c), c.Params("id"), domain.ProjectUpdate{
		Title: req.Title, Description: req.Description, KPIs: req.KPIs,
	})
	if err != nil {
		return err
	}
	return c.JSON(toProjectResponse(p))
}

// Cancel handles POST /api/projects/:id/cancel.
func (h *ProjectHandler) Cancel(c fiber.Ctx) error {
	p, err := h.projects.Cancel(c.Context(), middleware.UserID(c), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(toProjectResponse(p))
}

// ----- helpers shared by the handlers -----

// readJSON decodes the request body. A broken body, or a value of the
// wrong type (for example "budget": "abc"), becomes a 400 error.
func readJSON(c fiber.Ctx, dst any) error {
	if err := json.Unmarshal(c.Body(), dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field != "" {
			return &domain.ValidationError{Field: typeErr.Field, Message: "has the wrong type (" + typeErr.Value + ")"}
		}
		return fiber.NewError(fiber.StatusBadRequest, "request body must be valid JSON")
	}
	return nil
}

// queryInt reads an optional whole number from the URL, e.g. ?limit=10.
// A missing value gives 0, which means "use the default".
func queryInt(c fiber.Ctx, name string) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, &domain.ValidationError{Field: name, Message: "must be a whole number"}
	}
	return n, nil
}
