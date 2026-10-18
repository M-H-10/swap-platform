package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// ReputationHandler exposes reviews and the public reputation profile.
type ReputationHandler struct {
	reputation *service.ReputationService
}

// NewReputationHandler creates the handler.
func NewReputationHandler(reputation *service.ReputationService) *ReputationHandler {
	return &ReputationHandler{reputation: reputation}
}

type reputationResponse struct {
	ScoreBP        int       `json:"score_bp"`
	CompletedCount int       `json:"completed_count"`
	FailedCount    int       `json:"failed_count"`
	CancelledCount int       `json:"cancelled_count"`
	SuccessRateBP  int       `json:"success_rate_bp"`
	AvgRatingX100  *int      `json:"avg_rating_x100"` // null = no reviews yet
	AvgBudget      int64     `json:"avg_budget"`      // kopecks
	TotalVolume    int64     `json:"total_volume"`    // kopecks
	TrustWeightBP  int       `json:"trust_weight_bp"`
	Reason         string    `json:"reason"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type historyPointResponse struct {
	ScoreBP   int       `json:"score_bp"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type authorResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type reviewResponse struct {
	ID        string         `json:"id"`
	SwapID    string         `json:"swap_id"`
	Author    authorResponse `json:"author"`
	Rating    int            `json:"rating"`
	Comment   string         `json:"comment"`
	CreatedAt time.Time      `json:"created_at"`
}

func toReviewResponse(rv domain.Review) reviewResponse {
	return reviewResponse{
		ID: rv.ID, SwapID: rv.SwapID,
		Author: authorResponse{ID: rv.AuthorID, Username: rv.AuthorUsername},
		Rating: rv.Rating, Comment: rv.Comment, CreatedAt: rv.CreatedAt,
	}
}

// AddReview handles POST /api/swaps/:id/review with {"rating": 5, "comment": "..."}.
func (h *ReputationHandler) AddReview(c fiber.Ctx) error {
	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := readJSON(c, &req); err != nil {
		return err
	}
	rv, err := h.reputation.AddReview(c.Context(), middleware.UserID(c), c.Params("id"),
		domain.ReviewInput{Rating: req.Rating, Comment: req.Comment})
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(toReviewResponse(rv))
}

// Profile handles GET /api/users/:id/reputation (public).
func (h *ReputationHandler) Profile(c fiber.Ctx) error {
	p, err := h.reputation.Profile(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	cur := p.Current

	history := make([]historyPointResponse, len(p.History))
	for i, r := range p.History {
		history[i] = historyPointResponse{ScoreBP: r.ScoreBP, Reason: r.Reason, CreatedAt: r.CreatedAt}
	}
	reviews := make([]reviewResponse, len(p.Reviews))
	for i, rv := range p.Reviews {
		reviews[i] = toReviewResponse(rv)
	}

	return c.JSON(fiber.Map{
		"user": toPublicUser(p.User),
		"reputation": reputationResponse{
			ScoreBP: cur.ScoreBP, CompletedCount: cur.CompletedCount, FailedCount: cur.FailedCount,
			CancelledCount: cur.CancelledCount, SuccessRateBP: cur.SuccessRateBP,
			AvgRatingX100: cur.AvgRatingX100, AvgBudget: cur.AvgBudget, TotalVolume: cur.TotalVolume,
			TrustWeightBP: cur.TrustWeightBP, Reason: cur.Reason, UpdatedAt: cur.CreatedAt,
		},
		"history": history,
		"reviews": reviews,
	})
}
