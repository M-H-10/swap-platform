package domain

import (
	"testing"
	"time"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name      string
		in        ScoreInput
		wantScore int
		wantProb  int
	}{
		{
			// New account, average project: only the risks pull it down.
			name: "new user, average project",
			in: ScoreInput{Budget: 100_000_00, DurationDays: 30, Complexity: 5, TeamSize: 2,
				ReputationBP: 5000, AccountAgeDays: 0},
			wantScore: 3600, // 5000 - 1000 (new account) - 400 (tight deadline)
			wantProb:  3740,
		},
		{
			name: "experienced user, realistic project",
			in: ScoreInput{Budget: 300_000_00, DurationDays: 60, Complexity: 3, TeamSize: 3,
				ReputationBP: 8000, SuccessRateBP: 9000, Completed: 9, Failed: 1, AccountAgeDays: 200},
			wantScore: 7950, // 5000 + 900 + 800 + 450 + 500 + 300
			wantProb:  7655,
		},
		{
			name: "everything bad never goes below 5%",
			in: ScoreInput{Budget: 100_00, DurationDays: 10, Complexity: 10, TeamSize: 50,
				ReputationBP: 0, SuccessRateBP: 0, Failed: 5, AccountAgeDays: 1},
			wantScore: 0,
			wantProb:  500,
		},
		{
			name: "everything good never goes above 95%",
			in: ScoreInput{Budget: 1_000_000_00, DurationDays: 30, Complexity: 1, TeamSize: 1,
				ReputationBP: 10000, SuccessRateBP: 10000, Completed: 20, AccountAgeDays: 400},
			wantScore: 10000,
			wantProb:  9500,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateScore(tc.in)
			if got.ScoreBP != tc.wantScore || got.SuccessProbabilityBP != tc.wantProb {
				t.Fatalf("score = %d, prob = %d; want %d and %d (factors: %+v)",
					got.ScoreBP, got.SuccessProbabilityBP, tc.wantScore, tc.wantProb, got.Factors)
			}
		})
	}
}

func TestScoreIsDeterministic(t *testing.T) {
	in := ScoreInput{Budget: 50_000_00, DurationDays: 14, Complexity: 6, TeamSize: 4, ReputationBP: 6500, AccountAgeDays: 90}
	first := CalculateScore(in)
	for i := 0; i < 100; i++ {
		if again := CalculateScore(in); again.ScoreBP != first.ScoreBP {
			t.Fatalf("run %d gave %d, first run gave %d", i, again.ScoreBP, first.ScoreBP)
		}
	}
}

func TestNoHistoryMeansNoSuccessRatePoints(t *testing.T) {
	// A success rate of 0 with no finished projects is "unknown", not "bad".
	if p := successRatePoints(0, 0); p != 0 {
		t.Fatalf("expected 0 points without history, got %d", p)
	}
	if p := successRatePoints(0, 4); p != -1000 {
		t.Fatalf("expected -1000 for 0%% success over 4 projects, got %d", p)
	}
}

func TestNewScoreInputAccountAge(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	in := NewScoreInput(ProjectInput{Budget: 1}, CreatorStats{AccountSince: now.AddDate(0, 0, -45)}, now)
	if in.AccountAgeDays != 45 {
		t.Fatalf("account age = %d days, want 45", in.AccountAgeDays)
	}
}
