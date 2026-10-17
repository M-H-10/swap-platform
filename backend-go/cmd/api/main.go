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
)

const version = "2.0.0-dev"

func main() {
	// 1. Read the settings.
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// 2. Create the Fiber application.
	app := fiber.New(fiber.Config{
		AppName:      "Swap Platform API " + version,
		ErrorHandler: handler.ErrorHandler,
	})

	// 3. Middleware first, then routes.
	middleware.Register(app, cfg.CORSOrigins)
	handler.RegisterRoutes(app, handler.NewHealthHandler(version))

	// 4. Stop cleanly on Ctrl+C: finish the requests in progress first.
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
