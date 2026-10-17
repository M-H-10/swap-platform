package handler

import "github.com/gofiber/fiber/v3"

// RegisterRoutes connects every URL to the function that handles it.
func RegisterRoutes(app *fiber.App, health *HealthHandler) {
	api := app.Group("/api")

	api.Get("/health", health.Check)

	// Anything that did not match a route above ends here.
	app.Use(func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "route not found")
	})
}
