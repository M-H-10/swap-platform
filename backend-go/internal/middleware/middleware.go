// Package middleware holds code that runs before every request.
package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Register adds the global middleware. The order matters: each request
// passes through them from top to bottom.
func Register(app *fiber.App, corsOrigins []string) {
	// 1. If any handler panics, turn it into a 500 error instead of
	//    crashing the whole server.
	app.Use(recover.New())

	// 2. Give every request a unique ID (header X-Request-ID) so one
	//    request can be traced through the logs.
	app.Use(requestid.New())

	// 3. Print one line per request: time, status, duration, method, path.
	app.Use(logger.New())

	// 4. Allow only our frontend to call the API from the browser.
	//    AllowCredentials is needed later for the refresh-token cookie.
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowCredentials: true,
	}))
}
