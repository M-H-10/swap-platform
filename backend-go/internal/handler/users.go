package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// UserHandler exposes user search and public profiles.
type UserHandler struct {
	users *service.UserService
}

// NewUserHandler creates the handler.
func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

// publicUserResponse is what anyone may see about a user: no email.
type publicUserResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	ReputationBP int       `json:"reputation_bp"`
	CreatedAt    time.Time `json:"created_at"`
}

func toPublicUser(u domain.User) publicUserResponse {
	return publicUserResponse{ID: u.ID, Username: u.Username, Role: string(u.Role),
		ReputationBP: u.ReputationBP, CreatedAt: u.CreatedAt}
}

// Search handles GET /api/users?q=ali&limit=10 (you are left out of the results).
func (h *UserHandler) Search(c fiber.Ctx) error {
	limit, err := queryInt(c, "limit")
	if err != nil {
		return err
	}
	users, err := h.users.Search(c.Context(), middleware.UserID(c), c.Query("q"), limit)
	if err != nil {
		return err
	}
	items := make([]publicUserResponse, len(users))
	for i, u := range users {
		items[i] = toPublicUser(u)
	}
	return c.JSON(fiber.Map{"items": items})
}

// Get handles GET /api/users/:id.
func (h *UserHandler) Get(c fiber.Ctx) error {
	u, err := h.users.Get(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(toPublicUser(u))
}
