package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// PredictionHandler shows the formula's estimate next to the model's one.
type PredictionHandler struct {
	predictions *service.PredictionService
}

// NewPredictionHandler creates the handler.
func NewPredictionHandler(predictions *service.PredictionService) *PredictionHandler {
	return &PredictionHandler{predictions: predictions}
}

type formulaEstimateResponse struct {
	SuccessProbabilityBP int `json:"success_probability_bp"`
}

type mlEstimateResponse struct {
	SuccessProbabilityBP int       `json:"success_probability_bp"`
	ModelVersion         string    `json:"model_version"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type comparisonResponse struct {
	ProjectID    string                  `json:"project_id"`
	Status       string                  `json:"status"` // "ready" or "pending"
	Formula      formulaEstimateResponse `json:"formula"`
	ML           *mlEstimateResponse     `json:"ml"`            // null while pending
	DifferenceBP *int                    `json:"difference_bp"` // ml - formula; null while pending
}

// Get handles GET /api/projects/:id/prediction (public).
func (h *PredictionHandler) Get(c fiber.Ctx) error {
	cmp, err := h.predictions.Compare(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	resp := comparisonResponse{
		ProjectID: cmp.Project.ID,
		Status:    "pending",
		Formula:   formulaEstimateResponse{SuccessProbabilityBP: cmp.Project.SuccessProbabilityBP},
	}
	if cmp.ML != nil {
		diff := cmp.ML.ProbabilityBP - cmp.Project.SuccessProbabilityBP
		resp.Status = "ready"
		resp.ML = &mlEstimateResponse{
			SuccessProbabilityBP: cmp.ML.ProbabilityBP,
			ModelVersion:         cmp.ML.ModelVersion,
			UpdatedAt:            cmp.ML.UpdatedAt,
		}
		resp.DifferenceBP = &diff
	}
	return c.JSON(resp)
}

// Request handles POST /api/projects/:id/prediction: ask the model again
// (for example for a project created while the ML service was off).
// 202 Accepted means "received; the answer comes later".
func (h *PredictionHandler) Request(c fiber.Ctx) error {
	if err := h.predictions.Request(c.Context(), c.Params("id")); err != nil {
		return err
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "pending"})
}
