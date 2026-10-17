package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"
)

// ErrorHandler turns every error returned by a handler into one JSON shape:
//
//	{"error": {"code": 404, "message": "route not found"}}
//
// so the frontend always knows where to find the message.
func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	var fe *fiber.Error
	if errors.As(err, &fe) {
		code = fe.Code
		message = fe.Message
	}

	// 5xx errors are our fault: log the real cause, but never send it to
	// the client (it may contain internal details).
	if code >= fiber.StatusInternalServerError {
		log.Printf("[ERROR] %s %s: %v", c.Method(), c.Path(), err)
	}

	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{"code": code, "message": message},
	})
}
