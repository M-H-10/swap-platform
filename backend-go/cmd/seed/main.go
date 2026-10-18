// Command seed fills an empty development database with demo data:
// 4 users, 7 projects and swaps in every state (completed, failed, active,
// proposed, cancelled), with real money movements and reviews.
//
// Usage (from the backend-go folder, after the migrations):
//
//	go run ./cmd/seed
//
// Everything goes through the services, exactly like requests from the
// website, so the scores, the wallets and the reputation are all real.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/configs"
	"github.com/M-H-10/swap-platform/backend-go/internal/app"
	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/queue"
	"github.com/M-H-10/swap-platform/backend-go/internal/repository"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// demoPassword is the password of every demo user. It is fine to write it
// here because the program refuses to run outside development mode.
const demoPassword = "demo12345"

func main() {
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	if cfg.AppEnv != "development" {
		log.Fatal("seed refuses to run: APP_ENV is not \"development\"")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	// With RabbitMQ the new projects are also sent to the ML model.
	var publisher service.Publisher
	if cfg.RabbitMQURL != "" {
		mq := queue.New(cfg.RabbitMQURL, domain.QueuePredictRequests, domain.QueuePredictResults)
		go mq.Run(ctx)
		if waitFor(mq.Connected, 10*time.Second) {
			publisher = mq
		} else {
			log.Println("RabbitMQ is not reachable: projects are created without ML predictions")
		}
	}

	s, err := app.NewServices(cfg, pool, publisher, app.BcryptCost)
	if err != nil {
		log.Fatalf("setup error: %v", err)
	}

	if err := seed(ctx, s); err != nil {
		var ce *domain.ConflictError
		if errors.As(err, &ce) {
			fmt.Println("The demo data is already there (" + ce.Message + "). Nothing was changed.")
			return
		}
		log.Fatalf("seed failed: %v", err)
	}

	fmt.Println()
	fmt.Println("Demo data created. Log in with any of these accounts:")
	for _, u := range demoUsers {
		fmt.Printf("  %-26s password: %s\n", u.Email, demoPassword)
	}
	time.Sleep(500 * time.Millisecond) // let the last ML messages leave
	os.Exit(0)
}

var demoUsers = []domain.RegisterInput{
	{Username: "alice_dev", Email: "alice@example.com", Role: domain.RoleDeveloper},
	{Username: "boris_design", Email: "boris@example.com", Role: domain.RoleDesigner},
	{Username: "carol_pm", Email: "carol@example.com", Role: domain.RoleManager},
	{Username: "dima_new", Email: "dima@example.com", Role: domain.RoleDeveloper},
}

// seed creates everything. It stops at the first error.
func seed(ctx context.Context, s *app.Services) error {
	// 1. Users, each with 1 000 000 rubles of demo money.
	ids := map[string]string{}
	for _, in := range demoUsers {
		in.Password = demoPassword
		res, err := s.Auth.Register(ctx, in)
		if err != nil {
			return err // the first user already exists -> ConflictError
		}
		ids[in.Username] = res.User.ID
		if _, err := s.Wallets.Deposit(ctx, res.User.ID, 1_000_000_00); err != nil {
			return err
		}
		fmt.Println("user    ", in.Username)
	}
	alice, boris, carol, dima := ids["alice_dev"], ids["boris_design"], ids["carol_pm"], ids["dima_new"]

	// 2. Projects. Budgets are in kopecks: 150_000_00 = 150 000 rubles.
	shop, err := project(ctx, s, alice, domain.ProjectInput{
		Title: "Online store for handmade goods", Category: domain.CategoryWeb,
		Description: "A small online store with a catalogue, a cart and online payment.",
		Budget:      150_000_00, DurationDays: 60, Complexity: 5, TeamSize: 2,
		KPIs: []domain.KPI{{Name: "Orders", Target: "100 orders per month", Weight: 60}, {Name: "Conversion", Target: "2%", Weight: 40}},
	})
	if err != nil {
		return err
	}
	fitness, err := project(ctx, s, carol, domain.ProjectInput{
		Title: "Fitness tracking app", Category: domain.CategoryMobile,
		Description: "A mobile app that tracks workouts and sends weekly progress reports.",
		Budget:      400_000_00, DurationDays: 45, Complexity: 8, TeamSize: 1,
	})
	if err != nil {
		return err
	}
	brand, err := project(ctx, s, boris, domain.ProjectInput{
		Title: "Brand identity for a coffee shop", Category: domain.CategoryDesign,
		Description: "Logo, colours, menu design and signs for a new coffee shop.",
		Budget:      80_000_00, DurationDays: 30, Complexity: 3, TeamSize: 2,
	})
	if err != nil {
		return err
	}
	landing, err := project(ctx, s, alice, domain.ProjectInput{
		Title: "Marketing landing page", Category: domain.CategoryMarketing,
		Description: "A landing page and an e-mail campaign for a product launch.",
		Budget:      60_000_00, DurationDays: 21, Complexity: 2, TeamSize: 2,
	})
	if err != nil {
		return err
	}
	crm, err := project(ctx, s, carol, domain.ProjectInput{
		Title: "CRM for a small clinic", Category: domain.CategoryWeb,
		Description: "Patient records, appointments and SMS reminders for a clinic.",
		Budget:      600_000_00, DurationDays: 120, Complexity: 7, TeamSize: 4,
	})
	if err != nil {
		return err
	}
	if _, err := project(ctx, s, dima, domain.ProjectInput{
		Title: "Telegram bot for a bakery", Category: domain.CategoryOther,
		Description: "A bot that takes pre-orders and tells customers when bread is ready.",
		Budget:      40_000_00, DurationDays: 14, Complexity: 3, TeamSize: 1,
	}); err != nil {
		return err
	}
	if _, err := project(ctx, s, boris, domain.ProjectInput{
		Title: "Portfolio website", Category: domain.CategoryDesign,
		Description: "A personal portfolio with case studies and a contact form.",
		Budget:      50_000_00, DurationDays: 20, Complexity: 2, TeamSize: 1,
	}); err != nil {
		return err
	}

	// 3. Swaps in every state.
	// completed: both say "success", then both write a review.
	sw, err := activeSwap(ctx, s, shop, alice, boris)
	if err != nil {
		return err
	}
	if err := bothReport(ctx, s, sw, alice, boris, domain.OutcomeSuccess); err != nil {
		return err
	}
	if err := review(ctx, s, sw, alice, 5, "Great design work, always on time."); err != nil {
		return err
	}
	if err := review(ctx, s, sw, boris, 4, "Clear tasks and fast feedback."); err != nil {
		return err
	}
	fmt.Println("swap     completed:", shop.Title)

	// failed: both say "failure", the guarantees are paid as penalties.
	sw, err = activeSwap(ctx, s, fitness, carol, alice)
	if err != nil {
		return err
	}
	if err := bothReport(ctx, s, sw, carol, alice, domain.OutcomeFailure); err != nil {
		return err
	}
	if err := review(ctx, s, sw, carol, 2, "The deadline was too short and we missed it."); err != nil {
		return err
	}
	fmt.Println("swap     failed:   ", fitness.Title)

	// active: accepted, and boris has already reported success.
	sw, err = activeSwap(ctx, s, brand, boris, carol)
	if err != nil {
		return err
	}
	if _, err := s.Swaps.ReportOutcome(ctx, boris, sw.ID, domain.OutcomeSuccess); err != nil {
		return err
	}
	fmt.Println("swap     active:   ", brand.Title)

	// proposed: waiting for dima to accept or reject.
	if _, err := s.Swaps.Propose(ctx, alice, domain.SwapProposal{ProjectID: landing.ID, PartnerID: dima}); err != nil {
		return err
	}
	fmt.Println("swap     proposed: ", landing.Title)

	// cancelled: carol proposed and then cancelled.
	sw, err = s.Swaps.Propose(ctx, carol, domain.SwapProposal{ProjectID: crm.ID, PartnerID: boris})
	if err != nil {
		return err
	}
	if _, err := s.Swaps.Cancel(ctx, carol, sw.ID); err != nil {
		return err
	}
	fmt.Println("swap     cancelled:", crm.Title)
	return nil
}

func project(ctx context.Context, s *app.Services, creatorID string, in domain.ProjectInput) (domain.Project, error) {
	p, _, err := s.Projects.Create(ctx, creatorID, in)
	if err == nil {
		fmt.Println("project ", p.Title)
	}
	return p, err
}

// activeSwap proposes a swap (a = project creator) and lets b accept it.
func activeSwap(ctx context.Context, s *app.Services, p domain.Project, a, b string) (domain.Swap, error) {
	sw, err := s.Swaps.Propose(ctx, a, domain.SwapProposal{ProjectID: p.ID, PartnerID: b})
	if err != nil {
		return domain.Swap{}, err
	}
	return s.Swaps.Accept(ctx, b, sw.ID)
}

func bothReport(ctx context.Context, s *app.Services, sw domain.Swap, a, b string, o domain.Outcome) error {
	if _, err := s.Swaps.ReportOutcome(ctx, a, sw.ID, o); err != nil {
		return err
	}
	_, err := s.Swaps.ReportOutcome(ctx, b, sw.ID, o)
	return err
}

func review(ctx context.Context, s *app.Services, sw domain.Swap, authorID string, rating int, comment string) error {
	_, err := s.Reputation.AddReview(ctx, authorID, sw.ID, domain.ReviewInput{Rating: rating, Comment: comment})
	return err
}

// waitFor checks ok() every 100 ms until it is true or the time is up.
func waitFor(ok func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
