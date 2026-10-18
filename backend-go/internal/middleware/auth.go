package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// userIDKey is where RequireAuth stores the user ID for the next handlers.
const userIDKey = "userID"

// TokenParser is anything that can turn an access token into a user ID.
// The service's TokenManager fits; tests can pass a fake.
type TokenParser interface {
	ParseAccessToken(token string) (string, error)
}

// RequireAuth only lets a request through if it carries a valid access
// token in the header:  Authorization: Bearer <token>
func RequireAuth(tokens TokenParser) fiber.Handler {
	return func(c fiber.Ctx) error {
		token, ok := strings.CutPrefix(c.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing access token")
		}

		userID, err := tokens.ParseAccessToken(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired access token")
		}

		c.Locals(userIDKey, userID)
		return c.Next()
	}
}

// UserID returns the ID that RequireAuth stored for this request.
func UserID(c fiber.Ctx) string {
	id, _ := c.Locals(userIDKey).(string)
	return id
}
