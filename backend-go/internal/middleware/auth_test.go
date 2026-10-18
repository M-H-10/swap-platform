package middleware

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// fakeParser accepts only the token "good" and says it belongs to "user-1".
type fakeParser struct{}

func (fakeParser) ParseAccessToken(token string) (string, error) {
	if token == "good" {
		return "user-1", nil
	}
	return "", errors.New("bad token")
}

func newApp() *fiber.App {
	app := fiber.New()
	app.Get("/private", RequireAuth(fakeParser{}), func(c fiber.Ctx) error {
		return c.SendString(UserID(c))
	})
	return app
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantBody   string
	}{
		{"no header", "", fiber.StatusUnauthorized, ""},
		{"not bearer", "Basic abc", fiber.StatusUnauthorized, ""},
		{"wrong token", "Bearer bad", fiber.StatusUnauthorized, ""},
		{"valid token", "Bearer good", fiber.StatusOK, "user-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/private", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			resp, err := newApp().Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if tc.wantBody != "" {
				body, _ := io.ReadAll(resp.Body)
				if string(body) != tc.wantBody {
					t.Fatalf("body = %q, want %q", body, tc.wantBody)
				}
			}
		})
	}
}
