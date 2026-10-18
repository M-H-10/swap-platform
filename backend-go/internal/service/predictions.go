package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// PredictionStore is everything the prediction service needs from the database.
type PredictionStore interface {
	GetProject(ctx context.Context, id string) (domain.Project, error)
	GetCreatorStats(ctx context.Context, userID string) (domain.CreatorStats, error)
	SavePrediction(ctx context.Context, res domain.PredictionResult) error
	GetPrediction(ctx context.Context, projectID string) (domain.Prediction, error)
}

// Publisher sends a message to a queue. The RabbitMQ client has this method.
type Publisher interface {
	Publish(ctx context.Context, queue string, msg any) error
}

// PredictionService asks the ML service for estimates and stores its answers.
type PredictionService struct {
	store     PredictionStore
	publisher Publisher // nil when RabbitMQ is not configured
	now       func() time.Time
}

// NewPredictionService builds the service. publisher may be nil.
func NewPredictionService(store PredictionStore, publisher Publisher) *PredictionService {
	return &PredictionService{store: store, publisher: publisher, now: time.Now}
}

// Request sends the project's features to the ML service. It does not wait
// for the answer: the answer arrives later through HandleResult.
func (s *PredictionService) Request(ctx context.Context, projectID string) error {
	if s.publisher == nil {
		return fmt.Errorf("%w: RABBITMQ_URL is not set", domain.ErrUnavailable)
	}
	if !domain.IsUUID(projectID) {
		return domain.ErrNotFound
	}
	p, err := s.store.GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	stats, err := s.store.GetCreatorStats(ctx, p.CreatorID)
	if err != nil {
		return err
	}
	// The same input the formula uses, so the two answers can be compared.
	in := domain.NewScoreInput(domain.ProjectInput{
		Budget: p.Budget, DurationDays: p.DurationDays, Complexity: p.Complexity, TeamSize: p.TeamSize,
	}, stats, s.now())

	return s.publisher.Publish(ctx, domain.QueuePredictRequests, domain.PredictionRequest{
		ProjectID: p.ID,
		Features:  domain.NewMLFeatures(in),
	})
}

// HandleResult is called for every message in the results queue.
func (s *PredictionService) HandleResult(ctx context.Context, body []byte) error {
	var res domain.PredictionResult
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("bad result message: %w", err)
	}
	if err := res.Validate(); err != nil {
		return fmt.Errorf("bad result message: %w", err)
	}
	return s.store.SavePrediction(ctx, res)
}

// Comparison is the formula's answer next to the model's answer.
// ML is nil while the ML service has not answered yet.
type Comparison struct {
	Project domain.Project
	ML      *domain.Prediction
}

// Compare returns both estimates of a project.
func (s *PredictionService) Compare(ctx context.Context, projectID string) (Comparison, error) {
	if !domain.IsUUID(projectID) {
		return Comparison{}, domain.ErrNotFound
	}
	p, err := s.store.GetProject(ctx, projectID)
	if err != nil {
		return Comparison{}, err
	}
	ml, err := s.store.GetPrediction(ctx, projectID)
	if errors.Is(err, domain.ErrNotFound) {
		return Comparison{Project: p}, nil // not ready yet
	}
	if err != nil {
		return Comparison{}, err
	}
	return Comparison{Project: p, ML: &ml}, nil
}
