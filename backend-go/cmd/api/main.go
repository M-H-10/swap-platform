// Command api starts the Swap Platform HTTP server.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/configs"
	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/handler"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/queue"
	"github.com/M-H-10/swap-platform/backend-go/internal/repository"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

const version = "2.0.0-dev"

// bcryptCost 12 means about 0.3 s per password hash: fast enough for a
// user, slow enough to make guessing millions of passwords impractical.
const bcryptCost = 12

func main() {
	// 1. Read the settings.
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// ctx is cancelled on Ctrl+C. Everything that runs in the background
	// (the RabbitMQ connection, the HTTP server) stops when it is cancelled.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Connect to PostgreSQL. The pool is closed when main() returns.
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()
	log.Println("connected to PostgreSQL")

	// 3. Build the layers from the inside out:
	//    repository (SQL) -> service (rules) -> handler (HTTP).
	users := repository.NewUserRepository(pool)
	tokens := service.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authService, err := service.NewAuthService(users, tokens, bcryptCost)
	if err != nil {
		log.Fatalf("auth service error: %v", err)
	}

	// RabbitMQ is optional: without RABBITMQ_URL the server works as before,
	// only the ML estimates are switched off. (A Go detail: the variable has
	// the interface type, so it stays a real nil when RabbitMQ is off.)
	var publisher service.Publisher
	var mq *queue.Client
	if cfg.RabbitMQURL != "" {
		mq = queue.New(cfg.RabbitMQURL, domain.QueuePredictRequests, domain.QueuePredictResults)
		publisher = mq
	} else {
		log.Println("RABBITMQ_URL is empty: ML predictions are switched off")
	}
	predictionService := service.NewPredictionService(repository.NewPredictionRepository(pool), publisher)
	if mq != nil {
		mq.Consume(domain.QueuePredictResults, predictionService.HandleResult)
		go mq.Run(ctx) // connects in the background and reconnects if needed
	}

	projectService := service.NewProjectService(repository.NewProjectRepository(pool), predictionService)
	userService := service.NewUserService(users)
	walletService := service.NewWalletService(repository.NewWalletRepository(pool))
	reputationService := service.NewReputationService(repository.NewReputationRepository(pool))
	swapService := service.NewSwapService(repository.NewSwapRepository(pool), reputationService)

	// 4. Create the Fiber application.
	app := fiber.New(fiber.Config{
		AppName:      "Swap Platform API " + version,
		ErrorHandler: handler.ErrorHandler,
	})

	// 5. Middleware first, then routes.
	middleware.Register(app, cfg.CORSOrigins)
	handler.RegisterRoutes(app, handler.Handlers{
		Health:      handler.NewHealthHandler(version, pool),
		Auth:        handler.NewAuthHandler(authService, cfg.CookieSecure),
		Projects:    handler.NewProjectHandler(projectService),
		Users:       handler.NewUserHandler(userService),
		Wallets:     handler.NewWalletHandler(walletService),
		Swaps:       handler.NewSwapHandler(swapService),
		Reputation:  handler.NewReputationHandler(reputationService),
		Predictions: handler.NewPredictionHandler(predictionService),
		RequireAuth: middleware.RequireAuth(tokens),
		DemoDeposit: cfg.AppEnv == "development",
	})

	// 6. Stop cleanly on Ctrl+C: finish the requests in progress first.
	log.Printf("environment: %s", cfg.AppEnv)
	err = app.Listen(cfg.Addr(), fiber.ListenConfig{
		GracefulContext: ctx,
		ShutdownTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
