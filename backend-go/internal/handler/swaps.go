package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// SwapHandler exposes the swap routes.
type SwapHandler struct {
	swaps *service.SwapService
}

// NewSwapHandler creates the handler.
func NewSwapHandler(swaps *service.SwapService) *SwapHandler {
	return &SwapHandler{swaps: swaps}
}

// ----- request and response shapes -----

type swapRequest struct {
	ProjectID  string `json:"project_id"`
	PartnerID  string `json:"partner_id"`
	PartyARole string `json:"party_a_role"` // optional
	PartyBRole string `json:"party_b_role"` // optional
}

func (r swapRequest) toProposal() domain.SwapProposal {
	return domain.SwapProposal{
		ProjectID: r.ProjectID, PartnerID: r.PartnerID,
		PartyARole: domain.Role(r.PartyARole), PartyBRole: domain.Role(r.PartyBRole),
	}
}

type partyResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type projectRefResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// termsResponse holds the numbers of the contract. All money is in kopecks.
type termsResponse struct {
	ShareABP             int                `json:"share_a_bp"`
	ShareBBP             int                `json:"share_b_bp"`
	EscrowAmount         int64              `json:"escrow_amount"`
	PenaltyA             int64              `json:"penalty_a"`
	PenaltyB             int64              `json:"penalty_b"`
	SuccessProbabilityBP int                `json:"success_probability_bp"`
	RiskA                domain.RiskProfile `json:"risk_a"`
	RiskB                domain.RiskProfile `json:"risk_b"`
}

type swapResponse struct {
	ID                string             `json:"id"`
	Project           projectRefResponse `json:"project"`
	PartyA            partyResponse      `json:"party_a"`
	PartyB            partyResponse      `json:"party_b"`
	Terms             termsResponse      `json:"terms"`
	Status            string             `json:"status"`
	OutcomeA          *domain.Outcome    `json:"outcome_a"`
	OutcomeB          *domain.Outcome    `json:"outcome_b"`
	CancelRequestedBy *string            `json:"cancel_requested_by"`
	AcceptedAt        *time.Time         `json:"accepted_at"`
	DeadlineAt        *time.Time         `json:"deadline_at"`
	ClosedAt          *time.Time         `json:"closed_at"`
	CreatedAt         time.Time          `json:"created_at"`
}

func toTermsResponse(t domain.SwapTerms) termsResponse {
	return termsResponse{
		ShareABP: t.ShareABP, ShareBBP: t.ShareBBP, EscrowAmount: t.EscrowAmount,
		PenaltyA: t.PenaltyA, PenaltyB: t.PenaltyB, SuccessProbabilityBP: t.SuccessProbabilityBP,
		RiskA: t.RiskA, RiskB: t.RiskB,
	}
}

func toSwapResponse(s domain.Swap) swapResponse {
	return swapResponse{
		ID:                s.ID,
		Project:           projectRefResponse{ID: s.ProjectID, Title: s.ProjectTitle},
		PartyA:            partyResponse{ID: s.PartyAID, Username: s.PartyAUsername, Role: string(s.PartyARole)},
		PartyB:            partyResponse{ID: s.PartyBID, Username: s.PartyBUsername, Role: string(s.PartyBRole)},
		Terms:             toTermsResponse(s.Terms),
		Status:            string(s.Status),
		OutcomeA:          s.OutcomeA,
		OutcomeB:          s.OutcomeB,
		CancelRequestedBy: s.CancelRequestedBy,
		AcceptedAt:        s.AcceptedAt,
		DeadlineAt:        s.DeadlineAt,
		ClosedAt:          s.ClosedAt,
		CreatedAt:         s.CreatedAt,
	}
}

// ----- handlers -----

// Calculate handles POST /api/swaps/calculate: the terms, nothing saved.
func (h *SwapHandler) Calculate(c fiber.Ctx) error {
	var req swapRequest
	if err := readJSON(c, &req); err != nil {
		return err
	}
	s, err := h.swaps.Calculate(c.Context(), middleware.UserID(c), req.toProposal())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"project": projectRefResponse{ID: s.ProjectID, Title: s.ProjectTitle},
		"party_a": partyResponse{ID: s.PartyAID, Username: s.PartyAUsername, Role: string(s.PartyARole)},
		"party_b": partyResponse{ID: s.PartyBID, Username: s.PartyBUsername, Role: string(s.PartyBRole)},
		"terms":   toTermsResponse(s.Terms),
	})
}

// Propose handles POST /api/swaps.
func (h *SwapHandler) Propose(c fiber.Ctx) error {
	var req swapRequest
	if err := readJSON(c, &req); err != nil {
		return err
	}
	s, err := h.swaps.Propose(c.Context(), middleware.UserID(c), req.toProposal())
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(toSwapResponse(s))
}

// List handles GET /api/swaps?status=active
func (h *SwapHandler) List(c fiber.Ctx) error {
	swaps, err := h.swaps.List(c.Context(), middleware.UserID(c), domain.SwapStatus(c.Query("status")))
	if err != nil {
		return err
	}
	items := make([]swapResponse, len(swaps))
	for i, s := range swaps {
		items[i] = toSwapResponse(s)
	}
	return c.JSON(fiber.Map{"items": items})
}

// Get handles GET /api/swaps/:id.
func (h *SwapHandler) Get(c fiber.Ctx) error {
	s, err := h.swaps.Get(c.Context(), middleware.UserID(c), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(toSwapResponse(s))
}

// Accept handles POST /api/swaps/:id/accept.
func (h *SwapHandler) Accept(c fiber.Ctx) error {
	return h.respond(c)(h.swaps.Accept(c.Context(), middleware.UserID(c), c.Params("id")))
}

// Reject handles POST /api/swaps/:id/reject.
func (h *SwapHandler) Reject(c fiber.Ctx) error {
	return h.respond(c)(h.swaps.Reject(c.Context(), middleware.UserID(c), c.Params("id")))
}

// Cancel handles POST /api/swaps/:id/cancel.
func (h *SwapHandler) Cancel(c fiber.Ctx) error {
	return h.respond(c)(h.swaps.Cancel(c.Context(), middleware.UserID(c), c.Params("id")))
}

// Outcome handles POST /api/swaps/:id/outcome with {"outcome": "success"}.
func (h *SwapHandler) Outcome(c fiber.Ctx) error {
	var req struct {
		Outcome string `json:"outcome"`
	}
	if err := readJSON(c, &req); err != nil {
		return err
	}
	outcome, err := domain.ParseOutcome(req.Outcome)
	if err != nil {
		return err
	}
	return h.respond(c)(h.swaps.ReportOutcome(c.Context(), middleware.UserID(c), c.Params("id"), outcome))
}

// respond turns the (swap, error) pair that every action returns into
// the HTTP answer, so the four action handlers stay one line long.
func (h *SwapHandler) respond(c fiber.Ctx) func(domain.Swap, error) error {
	return func(s domain.Swap, err error) error {
		if err != nil {
			return err
		}
		return c.JSON(toSwapResponse(s))
	}
}
