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
	Wallets     *WalletHandler
	Swaps       *SwapHandler
	Reputation  *ReputationHandler
	RequireAuth fiber.Handler // middleware that checks the access token
	DemoDeposit bool          // true only in development: allows free test money
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

	if h.Wallets != nil {
		wallet := api.Group("/wallet")
		wallet.Get("/", h.RequireAuth, h.Wallets.Get)
		wallet.Get("/transactions", h.RequireAuth, h.Wallets.Transactions)
		if h.DemoDeposit {
			wallet.Post("/deposit", h.RequireAuth, h.Wallets.Deposit)
		}
	}

	if h.Swaps != nil {
		swaps := api.Group("/swaps")
		swaps.Post("/calculate", h.RequireAuth, h.Swaps.Calculate) // terms without saving
		swaps.Post("/", h.RequireAuth, h.Swaps.Propose)            // party A (project creator)
		swaps.Get("/", h.RequireAuth, h.Swaps.List)                // my swaps
		swaps.Get("/:id", h.RequireAuth, h.Swaps.Get)              // parties only
		swaps.Post("/:id/accept", h.RequireAuth, h.Swaps.Accept)   // party B
		swaps.Post("/:id/reject", h.RequireAuth, h.Swaps.Reject)   // party B
		swaps.Post("/:id/cancel", h.RequireAuth, h.Swaps.Cancel)   // either party
		swaps.Post("/:id/outcome", h.RequireAuth, h.Swaps.Outcome) // either party
	}

	if h.Reputation != nil {
		api.Post("/swaps/:id/review", h.RequireAuth, h.Reputation.AddReview) // either party, after the end
		api.Get("/users/:id/reputation", h.Reputation.Profile)               // anyone can view
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
