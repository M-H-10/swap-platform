package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// fakePinger pretends to be the database. If err is nil the "database"
// is up; otherwise Ping returns that error.
type fakePinger struct{ err error }

func (f fakePinger) Ping(ctx context.Context) error { return f.err }

// newTestApp builds a small app with only our routes, no real server.
func newTestApp(db Pinger) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
	RegisterRoutes(app, NewHealthHandler("test", db))
	return app
}

// getJSON sends a GET request and decodes the JSON answer.
func getJSON(t *testing.T, app *fiber.App, path string) (int, map[string]any) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	return resp.StatusCode, body
}

func TestHealthReturnsOK(t *testing.T) {
	code, body := getJSON(t, newTestApp(fakePinger{}), "/api/health")

	if code != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if body["status"] != "ok" || body["database"] != "up" {
		t.Errorf("body = %v, want status ok and database up", body)
	}
	if body["version"] != "test" {
		t.Errorf(`version field = %v, want "test"`, body["version"])
	}
}

func TestHealthReportsDatabaseDown(t *testing.T) {
	db := fakePinger{err: errors.New("connection refused")}
	code, body := getJSON(t, newTestApp(db), "/api/health")

	if code != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", code)
	}
	if body["status"] != "degraded" || body["database"] != "down" {
		t.Errorf("body = %v, want status degraded and database down", body)
	}
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	code, body := getJSON(t, newTestApp(fakePinger{}), "/api/does-not-exist")

	if code != fiber.StatusNotFound {
		t.Fatalf("status = %d, want 404", code)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("body has no error object: %v", body)
	}
	if errObj["code"] != float64(404) || errObj["message"] != "route not found" {
		t.Errorf("error = %v, want code 404 and message \"route not found\"", errObj)
	}
}
