package domain

import (
	"errors"
	"testing"
	"time"
)

var repNow = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

func daysAgo(d int) time.Time { return repNow.AddDate(0, 0, -d) }

func TestCalculateReputation(t *testing.T) {
	newAuthor := ReviewFact{Rating: 5, AuthorSince: daysAgo(0), AuthorCompleted: 1}
	trustedAuthor := ReviewFact{Rating: 5, AuthorSince: daysAgo(200), AuthorCompleted: 8}

	tests := []struct {
		name  string
		facts ReputationFacts
		want  int
	}{
		{"new account stays at 5000", ReputationFacts{AccountSince: daysAgo(0)}, 5000},
		// success (10000+10000)/3 = 6666, rating 5000, experience 5500
		{"one successful swap", ReputationFacts{Completed: 1, AccountSince: daysAgo(0)}, 5933},
		// success (0+10000)/3 = 3333
		{"one failed swap", ReputationFacts{Failed: 1, AccountSince: daysAgo(0)}, 4266},
		// a 5-star review from a brand new account (trust 10%) barely moves it
		{"success + review from new account", ReputationFacts{Completed: 1, AccountSince: daysAgo(0),
			Reviews: []ReviewFact{newAuthor}}, 6000},
		// the same 5 stars from a trusted account count 10 times more
		{"success + review from trusted account", ReputationFacts{Completed: 1, AccountSince: daysAgo(0),
			Reviews: []ReviewFact{trustedAuthor}}, 6428},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateReputation(tc.facts, repNow, "test")
			if got.ScoreBP != tc.want {
				t.Fatalf("score = %d, want %d", got.ScoreBP, tc.want)
			}
		})
	}
}

func TestReputationRawMetrics(t *testing.T) {
	facts := ReputationFacts{
		Completed: 3, Failed: 1, Cancelled: 2, TotalVolume: 400_000_00, AccountSince: daysAgo(10),
		Reviews: []ReviewFact{{Rating: 4, AuthorSince: daysAgo(200), AuthorCompleted: 8}},
	}
	rep := CalculateReputation(facts, repNow, "swap_completed")
	if rep.SuccessRateBP != 7500 || rep.AvgBudget != 100_000_00 || rep.CancelledCount != 2 {
		t.Fatalf("unexpected metrics: %+v", rep)
	}
	if rep.AvgRatingX100 == nil || *rep.AvgRatingX100 != 400 {
		t.Fatalf("average rating = %v, want 400", rep.AvgRatingX100)
	}
	if rep.Reason != "swap_completed" {
		t.Fatalf("reason = %q", rep.Reason)
	}
}

func TestNoReviewsMeansNoAverage(t *testing.T) {
	rep := CalculateReputation(ReputationFacts{AccountSince: daysAgo(5)}, repNow, "test")
	if rep.AvgRatingX100 != nil {
		t.Fatalf("expected nil average, got %d", *rep.AvgRatingX100)
	}
}

func TestTrustWeight(t *testing.T) {
	tests := []struct{ age, completed, want int }{
		{0, 0, 1000},   // brand new: the minimum
		{30, 0, 1999},  // 60% x 3333 (integer division drops the fraction)
		{45, 0, 3000},  // 60% x 5000
		{0, 5, 4000},   // 40% x 10000
		{90, 5, 10000}, // fully trusted
		{400, 50, 10000},
	}
	for _, tc := range tests {
		if got := TrustWeight(tc.age, tc.completed); got != tc.want {
			t.Fatalf("TrustWeight(%d, %d) = %d, want %d", tc.age, tc.completed, got, tc.want)
		}
	}
}

func TestFakeReviewsHaveLittleEffect(t *testing.T) {
	// Ten 5-star reviews from ten brand new accounts weigh as much as ONE
	// review from a trusted account.
	fakes := make([]ReviewFact, 10)
	for i := range fakes {
		fakes[i] = ReviewFact{Rating: 5, AuthorSince: daysAgo(0)}
	}
	tenFakes := CalculateReputation(ReputationFacts{AccountSince: daysAgo(0), Reviews: fakes}, repNow, "test")
	oneTrusted := CalculateReputation(ReputationFacts{AccountSince: daysAgo(0),
		Reviews: []ReviewFact{{Rating: 5, AuthorSince: daysAgo(200), AuthorCompleted: 8}}}, repNow, "test")
	if tenFakes.ScoreBP != oneTrusted.ScoreBP {
		t.Fatalf("10 fake reviews = %d, 1 trusted review = %d", tenFakes.ScoreBP, oneTrusted.ScoreBP)
	}
}

func TestReviewInputValidate(t *testing.T) {
	var ve *ValidationError
	if err := (ReviewInput{Rating: 0}).Validate(); !errors.As(err, &ve) || ve.Field != "rating" {
		t.Fatalf("expected rating error, got %v", err)
	}
	if err := (ReviewInput{Rating: 5, Comment: "great partner"}).Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}
