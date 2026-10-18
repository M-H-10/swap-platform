package handler

import (
	"bufio"
	"bytes"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// specOperations reads openapi.yaml and returns "METHOD /api/path" for
// every operation, with {id} written the Fiber way (:id).
func specOperations(t *testing.T) map[string]bool {
	t.Helper()
	pathLine := regexp.MustCompile(`^  (/\S*):$`)                         // "  /swaps/{id}:"
	methodLine := regexp.MustCompile(`^    (get|post|put|patch|delete):`) // "    post:"
	param := regexp.MustCompile(`\{(\w+)\}`)

	ops := map[string]bool{}
	current := ""
	scanner := bufio.NewScanner(bytes.NewReader(openapiSpec))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r") // Windows line endings
		if m := pathLine.FindStringSubmatch(line); m != nil {
			current = "/api" + param.ReplaceAllString(m[1], ":$1")
			continue
		}
		if strings.HasPrefix(line, "components:") {
			break
		}
		if m := methodLine.FindStringSubmatch(line); m != nil && current != "" {
			ops[strings.ToUpper(m[1])+" "+current] = true
		}
	}
	return ops
}

// TestEveryRouteIsDocumented fails when someone adds a route and forgets
// to describe it in openapi.yaml (or describes a route that does not exist).
func TestEveryRouteIsDocumented(t *testing.T) {
	app := fiber.New()
	noop := func(c fiber.Ctx) error { return nil }
	RegisterRoutes(app, Handlers{
		Health: &HealthHandler{}, Docs: NewDocsHandler(), Auth: &AuthHandler{},
		Projects: &ProjectHandler{}, Users: &UserHandler{}, Wallets: &WalletHandler{},
		Swaps: &SwapHandler{}, Reputation: &ReputationHandler{}, Predictions: &PredictionHandler{},
		RequireAuth: noop, DemoDeposit: true,
	})

	spec := specOperations(t)
	routes := map[string]bool{}
	for _, r := range app.GetRoutes(true) {
		if r.Method == fiber.MethodHead || strings.HasPrefix(r.Path, "/api/docs") || !strings.HasPrefix(r.Path, "/api/") {
			continue
		}
		// Group routes like projects.Get("/") end with "/"; Fiber also
		// answers them without it, and that is how the spec writes them.
		routes[r.Method+" "+strings.TrimSuffix(r.Path, "/")] = true
	}

	for op := range routes {
		if !spec[op] {
			t.Errorf("route %s is missing from openapi.yaml", op)
		}
	}
	for op := range spec {
		if !routes[op] {
			t.Errorf("openapi.yaml describes %s, but there is no such route", op)
		}
	}
	if len(routes) == 0 {
		t.Fatal("no routes found")
	}
}

func TestDocsPages(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app, Handlers{Health: &HealthHandler{}, Docs: NewDocsHandler()})

	for path, want := range map[string]string{
		"/api/docs":              "swagger-ui",
		"/api/docs/openapi.yaml": "openapi: 3.0.3",
	} {
		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != fiber.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("GET %s: status %d, body does not contain %q", path, resp.StatusCode, want)
		}
	}
}
