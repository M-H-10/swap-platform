package app_test

// Integration test: the real server (all layers) against a real
// PostgreSQL database. It runs only when TEST_DATABASE_URL is set, so a
// plain "go test ./..." without a database still passes:
//
//	TEST_DATABASE_URL=postgres://user:pass@localhost:5433/swap_test?sslmode=disable go test ./internal/app/
//
// Use a separate database for tests: the test creates users and projects.

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/M-H-10/swap-platform/backend-go/configs"
	"github.com/M-H-10/swap-platform/backend-go/internal/app"
	"github.com/M-H-10/swap-platform/backend-go/internal/repository"
	"github.com/M-H-10/swap-platform/backend-go/migrations"
)

// newTestServer migrates the test database and builds the whole app.
func newTestServer(t *testing.T) *fiber.App {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set: skipping the integration test")
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, strings.Replace(dbURL, "postgres://", "pgx5://", 1))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate up: %v", err)
	}
	m.Close()

	pool, err := repository.NewPool(context.Background(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	cfg := &configs.Config{
		AppEnv: "development", CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret: "integration-test-secret-0123456789abcdef", AccessTokenTTL: 15 * time.Minute,
		RefreshTokenTTL: time.Hour,
	}
	// bcrypt cost 4 (the minimum) keeps the test fast. No RabbitMQ: nil publisher.
	services, err := app.NewServices(cfg, pool, nil, 4)
	if err != nil {
		t.Fatal(err)
	}
	return app.NewServer(cfg, pool, services)
}

// client is one logged-in user of the test.
type client struct {
	t      *testing.T
	app    *fiber.App
	id     string
	token  string
	cookie string // the refresh_token cookie
}

// call sends a request and decodes the JSON answer (if there is one).
func (c *client) call(method, path string, body any, wantStatus int) map[string]any {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.cookie != "" {
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: c.cookie})
	}
	resp, err := c.app.Test(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "refresh_token" {
			c.cookie = ck.Value
		}
	}
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d; body: %s", method, path, resp.StatusCode, wantStatus, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

// register creates a user with a unique name, so the test can run many
// times on the same database.
func register(t *testing.T, server *fiber.App, prefix, role string) *client {
	t.Helper()
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	name := prefix + "_" + hex.EncodeToString(b)

	c := &client{t: t, app: server}
	res := c.call("POST", "/api/auth/register", map[string]any{
		"username": name, "email": name + "@example.com", "password": "integration123", "role": role,
	}, fiber.StatusCreated)
	c.token = res["access_token"].(string)
	c.id = res["user"].(map[string]any)["id"].(string)
	return c
}

func num(m map[string]any, key string) int64 { return int64(m[key].(float64)) }

func TestFullSwapFlow(t *testing.T) {
	server := newTestServer(t)
	ali := register(t, server, "ali", "developer")
	mona := register(t, server, "mona", "designer")
	stranger := register(t, server, "eve", "other")

	// 1. Money: 100 000 rubles each.
	for _, c := range []*client{ali, mona} {
		w := c.call("POST", "/api/wallet/deposit", map[string]any{"amount": 100_000_00}, fiber.StatusOK)
		if num(w, "balance") != 100_000_00 {
			t.Fatalf("balance after deposit = %v", w["balance"])
		}
	}

	// 2. Ali creates a project; the formula scores it at once.
	res := ali.call("POST", "/api/projects", map[string]any{
		"title": "Integration shop", "description": "A project created by the integration test.",
		"category": "web", "budget": 150_000_00, "duration_days": 60, "complexity": 5, "team_size": 2,
	}, fiber.StatusCreated)
	project := res["project"].(map[string]any)
	projectID := project["id"].(string)
	if p := num(project, "success_probability_bp"); p < 500 || p > 9500 {
		t.Fatalf("probability %d outside 5%%..95%%", p)
	}

	// Without RabbitMQ the ML estimate stays "pending" and asking again is 503.
	pred := ali.call("GET", "/api/projects/"+projectID+"/prediction", nil, fiber.StatusOK)
	if pred["status"] != "pending" || pred["ml"] != nil {
		t.Fatalf("prediction = %v, want pending", pred)
	}
	ali.call("POST", "/api/projects/"+projectID+"/prediction", nil, fiber.StatusServiceUnavailable)

	// 3. Only the creator may propose; Ali proposes to Mona.
	proposal := map[string]any{"project_id": projectID, "partner_id": ali.id}
	mona.call("POST", "/api/swaps", proposal, fiber.StatusForbidden)
	proposal["partner_id"] = mona.id
	swap := ali.call("POST", "/api/swaps", proposal, fiber.StatusCreated)
	swapID := swap["id"].(string)
	terms := swap["terms"].(map[string]any)
	penaltyA, penaltyB := num(terms, "penalty_a"), num(terms, "penalty_b")

	// A stranger cannot even see that the swap exists.
	stranger.call("GET", "/api/swaps/"+swapID, nil, fiber.StatusNotFound)

	// 4. Mona accepts: both guarantees are frozen.
	swap = mona.call("POST", "/api/swaps/"+swapID+"/accept", nil, fiber.StatusOK)
	if swap["status"] != "active" {
		t.Fatalf("status after accept = %v", swap["status"])
	}
	w := ali.call("GET", "/api/wallet", nil, fiber.StatusOK)
	if num(w, "balance") != 100_000_00-penaltyA || num(w, "frozen_balance") != penaltyA {
		t.Fatalf("ali wallet after accept = %v", w)
	}

	// 5. Different answers change nothing; the same answers close the swap.
	ali.call("POST", "/api/swaps/"+swapID+"/outcome", map[string]any{"outcome": "success"}, fiber.StatusOK)
	swap = mona.call("POST", "/api/swaps/"+swapID+"/outcome", map[string]any{"outcome": "failure"}, fiber.StatusOK)
	if swap["status"] != "active" {
		t.Fatalf("status after disagreement = %v, want active", swap["status"])
	}
	swap = mona.call("POST", "/api/swaps/"+swapID+"/outcome", map[string]any{"outcome": "success"}, fiber.StatusOK)
	if swap["status"] != "completed" {
		t.Fatalf("status after agreement = %v, want completed", swap["status"])
	}

	// Each party got its guarantee back minus the 0.5% fee.
	for _, tc := range []struct {
		c       *client
		penalty int64
	}{{ali, penaltyA}, {mona, penaltyB}} {
		w := tc.c.call("GET", "/api/wallet", nil, fiber.StatusOK)
		want := 100_000_00 - tc.penalty*50/10000
		if num(w, "balance") != want || num(w, "frozen_balance") != 0 {
			t.Errorf("wallet after completion = %v, want balance %d and nothing frozen", w, want)
		}
	}

	// 6. One review per author per swap.
	ali.call("POST", "/api/swaps/"+swapID+"/review", map[string]any{"rating": 5, "comment": "Great"}, fiber.StatusCreated)
	ali.call("POST", "/api/swaps/"+swapID+"/review", map[string]any{"rating": 1}, fiber.StatusConflict)

	profile := stranger.call("GET", "/api/users/"+mona.id+"/reputation", nil, fiber.StatusOK)
	rep := profile["reputation"].(map[string]any)
	if num(rep, "completed_count") != 1 || len(profile["reviews"].([]any)) != 1 {
		t.Fatalf("mona's profile = %v", profile)
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	server := newTestServer(t)
	ali := register(t, server, "ali", "developer")
	oldCookie := ali.cookie

	// A refresh gives a new cookie...
	ali.call("POST", "/api/auth/refresh", nil, fiber.StatusOK)
	if ali.cookie == "" || ali.cookie == oldCookie {
		t.Fatal("refresh did not rotate the cookie")
	}
	newCookie := ali.cookie

	// ...and using the old one again is treated as theft: 401, and the
	// new one stops working too.
	ali.cookie = oldCookie
	ali.call("POST", "/api/auth/refresh", nil, fiber.StatusUnauthorized)
	ali.cookie = newCookie
	ali.call("POST", "/api/auth/refresh", nil, fiber.StatusUnauthorized)
}
