package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// ErrorHandler turns every error returned by a handler into one JSON shape:
//
//	{"error": {"code": 409, "message": "email is already registered"}}
//
// Validation errors also say which field is wrong:
//
//	{"error": {"code": 400, "message": "...", "field": "password"}}
func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"
	field := ""

	var fe *fiber.Error
	var ve *domain.ValidationError
	var ce *domain.ConflictError

	switch {
	case errors.As(err, &fe):
		code, message = fe.Code, fe.Message
	case errors.As(err, &ve):
		code, message, field = fiber.StatusBadRequest, ve.Error(), ve.Field
	case errors.As(err, &ce):
		code, message = fiber.StatusConflict, ce.Message
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken):
		code, message = fiber.StatusUnauthorized, err.Error()
	case errors.Is(err, domain.ErrInsufficientFunds):
		code, message = fiber.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrForbidden):
		code, message = fiber.StatusForbidden, err.Error()
	case errors.Is(err, domain.ErrNotFound):
		code, message = fiber.StatusNotFound, "not found"
	}

	// 5xx errors are our fault: log the real cause, but never send it to
	// the client (it may contain internal details).
	if code >= fiber.StatusInternalServerError {
		log.Printf("[ERROR] %s %s: %v", c.Method(), c.Path(), err)
	}

	body := fiber.Map{"code": code, "message": message}
	if field != "" {
		body["field"] = field
	}
	return c.Status(code).JSON(fiber.Map{"error": body})
}
