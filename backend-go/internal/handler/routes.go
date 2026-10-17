package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// Handlers groups everything RegisterRoutes needs. A nil field means
// "this part is not used" (handy in tests).
type Handlers struct {
	Health      *HealthHandler
	Auth        *AuthHandler
	Projects    *ProjectHandler
	Users       *UserHandler
	RequireAuth fiber.Handler // middleware that checks the access token
}

// RegisterRoutes connects every URL to the function that handles it.
func RegisterRoutes(app *fiber.App, h Handlers) {
	api := app.Group("/api")

	api.Get("/health", h.Health.Check)

	if h.Auth != nil {
		auth := api.Group("/auth")
		auth.Post("/register", authLimiter(), h.Auth.Register)
		auth.Post("/login", authLimiter(), h.Auth.Login)
		auth.Post("/refresh", h.Auth.Refresh)
		auth.Post("/logout", h.Auth.Logout)
		auth.Get("/me", h.RequireAuth, h.Auth.Me)
	}

	if h.Projects != nil {
		projects := api.Group("/projects")
		projects.Get("/", h.Projects.List)                             // anyone can browse
		projects.Post("/", h.RequireAuth, h.Projects.Create)           // logged-in users only
		projects.Post("/estimate", h.RequireAuth, h.Projects.Estimate) // score without saving
		projects.Get("/:id", h.Projects.Get)                           // anyone can view
		projects.Patch("/:id", h.RequireAuth, h.Projects.Update)       // creator only
		projects.Post("/:id/cancel", h.RequireAuth, h.Projects.Cancel) // creator only
	}

	if h.Users != nil {
		users := api.Group("/users")
		users.Get("/", h.RequireAuth, h.Users.Search)
		users.Get("/:id", h.Users.Get)
	}

	// Anything that did not match a route above ends here.
	app.Use(func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "route not found")
	})
}

// authLimiter allows at most 10 requests per minute from one IP address.
// It slows down password guessing on login and mass account creation on
// register. Each route gets its own counter.
func authLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        10,
		Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many attempts, try again in a minute")
		},
	})
}
