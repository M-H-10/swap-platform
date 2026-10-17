package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// newTestApp builds a small app with only our routes, no real server.
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	RegisterRoutes(app, NewHealthHandler("test"))
	return app
}

func TestHealthReturnsOK(t *testing.T) {
	app := newTestApp()

	resp, err := app.Test(httptest.NewRequest("GET", "/api/health", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`status field = %v, want "ok"`, body["status"])
	}
	if body["version"] != "test" {
		t.Errorf(`version field = %v, want "test"`, body["version"])
	}
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	app := newTestApp()

	resp, err := app.Test(httptest.NewRequest("GET", "/api/does-not-exist", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}

	var body struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body.Error.Code != 404 || body.Error.Message != "route not found" {
		t.Errorf("error = %+v, want code 404 and message \"route not found\"", body.Error)
	}
}
