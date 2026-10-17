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
	"github.com/M-H-10/swap-platform/backend-go/internal/handler"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/repository"
)

const version = "2.0.0-dev"

func main() {
	// 1. Read the settings.
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// 2. Connect to PostgreSQL. The pool is closed when main() returns.
	pool, err := repository.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()
	log.Println("connected to PostgreSQL")

	// 3. Create the Fiber application.
	app := fiber.New(fiber.Config{
		AppName:      "Swap Platform API " + version,
		ErrorHandler: handler.ErrorHandler,
	})

	// 4. Middleware first, then routes.
	middleware.Register(app, cfg.CORSOrigins)
	handler.RegisterRoutes(app, handler.NewHealthHandler(version, pool))

	// 5. Stop cleanly on Ctrl+C: finish the requests in progress first.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
