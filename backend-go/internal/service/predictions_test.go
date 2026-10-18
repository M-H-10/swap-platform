package service

import (
	"context"
	"errors"
	"testing"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// fakePredictionStore reuses the fake project store and adds the
// predictions table as a map.
type fakePredictionStore struct {
	*fakeProjectStore
	saved map[string]domain.PredictionResult
}

func (f *fakePredictionStore) SavePrediction(_ context.Context, res domain.PredictionResult) error {
	if _, ok := f.projects[res.ProjectID]; !ok {
		return domain.ErrNotFound
	}
	f.saved[res.ProjectID] = res
	return nil
}

func (f *fakePredictionStore) GetPrediction(_ context.Context, id string) (domain.Prediction, error) {
	r, ok := f.saved[id]
	if !ok {
		return domain.Prediction{}, domain.ErrNotFound
	}
	return domain.Prediction{ProjectID: id, ProbabilityBP: r.ProbabilityBP, ModelVersion: r.ModelVersion}, nil
}

// fakePublisher remembers the messages instead of sending them to RabbitMQ.
type fakePublisher struct {
	queue    string
	messages []domain.PredictionRequest
}

func (p *fakePublisher) Publish(_ context.Context, queue string, msg any) error {
	p.queue = queue
	p.messages = append(p.messages, msg.(domain.PredictionRequest))
	return nil
}

// newPredictionTestServices connects a project service and a prediction
// service to the same fake database, like main.go does with the real one.
func newPredictionTestServices() (*ProjectService, *PredictionService, *fakePredictionStore, *fakePublisher) {
	projects, store := newProjectTestService()
	pstore := &fakePredictionStore{fakeProjectStore: store, saved: map[string]domain.PredictionResult{}}
	pub := &fakePublisher{}
	predictions := NewPredictionService(pstore, pub)
	predictions.now = projects.now
	projects.predictions = predictions
	return projects, predictions, pstore, pub
}

func TestCreatingAProjectAsksTheModel(t *testing.T) {
	projects, _, _, pub := newPredictionTestServices()
	p, _, err := projects.Create(context.Background(), "owner", shopInput())
	if err != nil {
		t.Fatal(err)
	}
	if pub.queue != domain.QueuePredictRequests || len(pub.messages) != 1 {
		t.Fatalf("expected one message in %s, got %d in %q", domain.QueuePredictRequests, len(pub.messages), pub.queue)
	}
	msg := pub.messages[0]
	if msg.ProjectID != p.ID || msg.Features.Budget != 100_000_00 || msg.Features.ReputationBP != 5000 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func TestResultIsSavedAndCompared(t *testing.T) {
	projects, predictions, _, _ := newPredictionTestServices()
	ctx := context.Background()
	p, _, _ := projects.Create(ctx, "owner", shopInput())

	cmp, err := predictions.Compare(ctx, p.ID)
	if err != nil || cmp.ML != nil {
		t.Fatalf("before the answer ML must be nil, got %+v (%v)", cmp.ML, err)
	}

	body := []byte(`{"project_id":"` + p.ID + `","probability_bp":6100,"model_version":"rf-1"}`)
	if err := predictions.HandleResult(ctx, body); err != nil {
		t.Fatal(err)
	}
	cmp, _ = predictions.Compare(ctx, p.ID)
	if cmp.ML == nil || cmp.ML.ProbabilityBP != 6100 || cmp.Project.SuccessProbabilityBP == 0 {
		t.Fatalf("unexpected comparison: %+v", cmp)
	}
}

func TestBadResultsAreRejected(t *testing.T) {
	_, predictions, _, _ := newPredictionTestServices()
	for _, body := range []string{
		`not json`,
		`{"project_id":"x","probability_bp":5000,"model_version":"rf-1"}`,
		`{"project_id":"00000000-0000-0000-0000-000000000001","probability_bp":20000,"model_version":"rf-1"}`,
	} {
		if err := predictions.HandleResult(context.Background(), []byte(body)); err == nil {
			t.Fatalf("expected an error for %s", body)
		}
	}
}

func TestWithoutRabbitMQTheRequestIsUnavailable(t *testing.T) {
	svc := NewPredictionService(&fakePredictionStore{fakeProjectStore: newFakeProjectStore()}, nil)
	err := svc.Request(context.Background(), "00000000-0000-0000-0000-000000000001")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}
