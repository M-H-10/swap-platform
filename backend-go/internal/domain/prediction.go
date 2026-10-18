package domain

import (
	"errors"
	"time"
)

// ErrUnavailable means a helper service (here: the ML service behind
// RabbitMQ) cannot be reached right now. The request may work later.
var ErrUnavailable = errors.New("service temporarily unavailable")

// The two RabbitMQ queues shared by Go and Python. Go writes requests to
// the first one; Python reads them and writes answers to the second one.
const (
	QueuePredictRequests = "ml.predict"
	QueuePredictResults  = "ml.result"
)

// MLFeatures is what the model looks at. It holds the same facts as the
// formula's ScoreInput, with JSON names that the Python code reads.
type MLFeatures struct {
	Budget         int64 `json:"budget"` // kopecks
	DurationDays   int   `json:"duration_days"`
	Complexity     int   `json:"complexity"`
	TeamSize       int   `json:"team_size"`
	ReputationBP   int   `json:"reputation_bp"`
	SuccessRateBP  int   `json:"success_rate_bp"`
	Completed      int   `json:"completed"`
	Failed         int   `json:"failed"`
	AccountAgeDays int   `json:"account_age_days"`
}

// NewMLFeatures copies the formula's input into the message format.
func NewMLFeatures(in ScoreInput) MLFeatures {
	return MLFeatures{
		Budget: in.Budget, DurationDays: in.DurationDays, Complexity: in.Complexity,
		TeamSize: in.TeamSize, ReputationBP: in.ReputationBP, SuccessRateBP: in.SuccessRateBP,
		Completed: in.Completed, Failed: in.Failed, AccountAgeDays: in.AccountAgeDays,
	}
}

// PredictionRequest is the message Go sends to the ML service.
type PredictionRequest struct {
	ProjectID string     `json:"project_id"`
	Features  MLFeatures `json:"features"`
}

// PredictionResult is the message the ML service sends back.
type PredictionResult struct {
	ProjectID     string `json:"project_id"`
	ProbabilityBP int    `json:"probability_bp"`
	ModelVersion  string `json:"model_version"`
}

// Validate checks a message that came from outside the Go program.
func (r PredictionResult) Validate() error {
	if !IsUUID(r.ProjectID) {
		return &ValidationError{"project_id", "must be a project id"}
	}
	if r.ProbabilityBP < 0 || r.ProbabilityBP > 10000 {
		return &ValidationError{"probability_bp", "must be 0-10000"}
	}
	if r.ModelVersion == "" || len(r.ModelVersion) > 50 {
		return &ValidationError{"model_version", "must be 1-50 characters"}
	}
	return nil
}

// Prediction is a saved ML estimate of one project.
type Prediction struct {
	ProjectID     string
	ProbabilityBP int
	ModelVersion  string
	UpdatedAt     time.Time
}
