// Package handler holds the HTTP layer: it reads requests and writes JSON.
package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

// HealthHandler answers "is the server alive?".
type HealthHandler struct {
	version   string
	startedAt time.Time
}

// NewHealthHandler creates the handler and remembers when the server started.
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version, startedAt: time.Now()}
}

// Check handles GET /api/health.
func (h *HealthHandler) Check(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":         "ok",
		"service":        "swap-platform-api",
		"version":        h.version,
		"uptime_seconds": int(time.Since(h.startedAt).Seconds()),
		"time":           time.Now().UTC().Format(time.RFC3339),
	})
}
