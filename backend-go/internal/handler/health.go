// Package handler holds the HTTP layer: it reads requests and writes JSON.
package handler

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Pinger is anything that can answer "are you alive?".
// The PostgreSQL pool has a Ping method, so it fits this interface,
// and in tests we can pass a fake one instead of a real database.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler answers "is the server alive, and can it reach the database?".
type HealthHandler struct {
	version   string
	startedAt time.Time
	db        Pinger
}

// NewHealthHandler creates the handler and remembers when the server started.
func NewHealthHandler(version string, db Pinger) *HealthHandler {
	return &HealthHandler{version: version, startedAt: time.Now(), db: db}
}

// Check handles GET /api/health.
func (h *HealthHandler) Check(c fiber.Ctx) error {
	status, dbStatus, code := "ok", "up", fiber.StatusOK

	// Never let a slow database make this request hang.
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		log.Printf("[WARN] health: database ping failed: %v", err)
		status, dbStatus, code = "degraded", "down", fiber.StatusServiceUnavailable
	}

	return c.Status(code).JSON(fiber.Map{
		"status":         status,
		"service":        "swap-platform-api",
		"version":        h.version,
		"database":       dbStatus,
		"uptime_seconds": int(time.Since(h.startedAt).Seconds()),
		"time":           time.Now().UTC().Format(time.RFC3339),
	})
}
