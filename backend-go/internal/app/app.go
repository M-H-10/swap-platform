// Package app builds the whole program from its parts: repositories
// (SQL) -> services (rules) -> handlers (HTTP). The API server, the seed
// command and the integration test all use it, so the layers are
// connected in exactly one place.
package app

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/M-H-10/swap-platform/backend-go/configs"
	"github.com/M-H-10/swap-platform/backend-go/internal/handler"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/repository"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// Version is shown by GET /api/health and in the Fiber start-up banner.
const Version = "2.0.0"

// BcryptCost 12 means about 0.3 s per password hash: fast enough for a
// user, slow enough to make guessing millions of passwords impractical.
const BcryptCost = 12

// Services holds every service of the application.
type Services struct {
	Tokens      *service.TokenManager
	Auth        *service.AuthService
	Users       *service.UserService
	Projects    *service.ProjectService
	Wallets     *service.WalletService
	Swaps       *service.SwapService
	Reputation  *service.ReputationService
	Predictions *service.PredictionService
}

// NewServices connects the repositories to the services.
// publisher may be nil: then ML predictions are switched off.
func NewServices(cfg *configs.Config, pool *pgxpool.Pool, publisher service.Publisher, bcryptCost int) (*Services, error) {
	users := repository.NewUserRepository(pool)
	tokens := service.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	auth, err := service.NewAuthService(users, tokens, bcryptCost)
	if err != nil {
		return nil, err
	}

	predictions := service.NewPredictionService(repository.NewPredictionRepository(pool), publisher)
	reputation := service.NewReputationService(repository.NewReputationRepository(pool))

	return &Services{
		Tokens:      tokens,
		Auth:        auth,
		Users:       service.NewUserService(users),
		Projects:    service.NewProjectService(repository.NewProjectRepository(pool), predictions),
		Wallets:     service.NewWalletService(repository.NewWalletRepository(pool)),
		Swaps:       service.NewSwapService(repository.NewSwapRepository(pool), reputation),
		Reputation:  reputation,
		Predictions: predictions,
	}, nil
}

// NewServer creates the Fiber application with middleware and all routes.
func NewServer(cfg *configs.Config, pool *pgxpool.Pool, s *Services) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Swap Platform API " + Version,
		ErrorHandler: handler.ErrorHandler,
	})

	// Middleware first, then routes.
	middleware.Register(app, cfg.CORSOrigins)
	handler.RegisterRoutes(app, handler.Handlers{
		Health:      handler.NewHealthHandler(Version, pool),
		Docs:        handler.NewDocsHandler(),
		Auth:        handler.NewAuthHandler(s.Auth, cfg.CookieSecure),
		Projects:    handler.NewProjectHandler(s.Projects),
		Users:       handler.NewUserHandler(s.Users),
		Wallets:     handler.NewWalletHandler(s.Wallets),
		Swaps:       handler.NewSwapHandler(s.Swaps),
		Reputation:  handler.NewReputationHandler(s.Reputation),
		Predictions: handler.NewPredictionHandler(s.Predictions),
		RequireAuth: middleware.RequireAuth(s.Tokens),
		DemoDeposit: cfg.AppEnv == "development",
	})
	return app
}
