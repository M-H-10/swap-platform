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
	"github.com/M-H-10/swap-platform/backend-go/internal/app"
	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/queue"
	"github.com/M-H-10/swap-platform/backend-go/internal/repository"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

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

	// 3. RabbitMQ is optional: without RABBITMQ_URL the server works as
	// before, only the ML estimates are switched off. (A Go detail: the
	// variable has the interface type, so it stays a real nil when off.)
	var publisher service.Publisher
	var mq *queue.Client
	if cfg.RabbitMQURL != "" {
		mq = queue.New(cfg.RabbitMQURL, domain.QueuePredictRequests, domain.QueuePredictResults)
		publisher = mq
	} else {
		log.Println("RABBITMQ_URL is empty: ML predictions are switched off")
	}

	// 4. All layers: repository (SQL) -> service (rules) -> handler (HTTP).
	services, err := app.NewServices(cfg, pool, publisher, app.BcryptCost)
	if err != nil {
		log.Fatalf("setup error: %v", err)
	}
	if mq != nil {
		mq.Consume(domain.QueuePredictResults, services.Predictions.HandleResult)
		go mq.Run(ctx) // connects in the background and reconnects if needed
	}
	server := app.NewServer(cfg, pool, services)

	// 5. Stop cleanly on Ctrl+C: finish the requests in progress first.
	log.Printf("environment: %s", cfg.AppEnv)
	err = server.Listen(cfg.Addr(), fiber.ListenConfig{
		GracefulContext: ctx,
		ShutdownTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
