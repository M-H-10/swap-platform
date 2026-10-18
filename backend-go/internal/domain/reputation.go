package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Reputation is one snapshot of a user's reputation. Every recalculation
// adds a new snapshot; the newest one is the current reputation.
type Reputation struct {
	ScoreBP        int    // 0..10000, the number everybody sees
	CompletedCount int    // swaps that ended in success
	FailedCount    int    // swaps that ended in failure
	CancelledCount int    // active swaps cancelled by both parties
	SuccessRateBP  int    // completed / (completed + failed), 0 if none
	AvgRatingX100  *int   // weighted average review, 100..500; nil = no reviews
	AvgBudget      int64  // kopecks, average budget of finished projects
	TotalVolume    int64  // kopecks, sum of budgets of finished projects
	TrustWeightBP  int    // how much this user's reviews count (anti-fraud)
	Reason         string // why it was recalculated
	CreatedAt      time.Time
}

// ReviewFact is one review as the formula sees it: the stars, and how far
// the author can be trusted.
type ReviewFact struct {
	Rating          int
	AuthorSince     time.Time
	AuthorCompleted int
}

// ReputationFacts is everything the formula looks at, read from the database.
type ReputationFacts struct {
	Completed    int
	Failed       int
	Cancelled    int
	TotalVolume  int64
	AccountSince time.Time
	Reviews      []ReviewFact
}

// The reputation formula (all integers, basis points):
//
//	score = 50% success + 30% rating + 20% experience
//
// Each part starts at a neutral 5000 and moves only as real history comes
// in, so a new account starts exactly at 5000 (the same as at registration).
//   - success:    success rate, pulled towards 50% as if there were 2
//     extra "average" projects (one lucky project is not 100%).
//   - rating:     reviews weighted by the AUTHOR's trust, pulled towards
//     3 stars as if there were 2 extra reviews from trusted users.
//   - experience: 5000 + 500 per finished project, up to 10 projects.
const (
	priorProjects        = 2
	priorReviewWeight    = 2 * 10000 // two reviews from fully trusted users
	neutralRatingX100    = 300       // 3 stars
	minTrustBP           = 1000      // even a brand new account counts a little
	trustFullAgeDays     = 90
	trustFullCompletedAt = 5
)

// CalculateReputation turns the facts into a snapshot. now is passed in so
// the same facts always give the same result.
func CalculateReputation(f ReputationFacts, now time.Time, reason string) Reputation {
	finished := f.Completed + f.Failed

	rep := Reputation{
		CompletedCount: f.Completed,
		FailedCount:    f.Failed,
		CancelledCount: f.Cancelled,
		TotalVolume:    f.TotalVolume,
		TrustWeightBP:  TrustWeight(ageInDays(f.AccountSince, now), f.Completed),
		Reason:         reason,
	}
	if finished > 0 {
		rep.SuccessRateBP = f.Completed * 10000 / finished
		rep.AvgBudget = f.TotalVolume / int64(finished)
	}

	// Weighted average of the reviews: a review from a trusted author
	// counts up to 10 times more than one from a brand new account.
	var sumWeighted, sumWeights int64
	for _, r := range f.Reviews {
		w := int64(TrustWeight(ageInDays(r.AuthorSince, now), r.AuthorCompleted))
		sumWeighted += int64(r.Rating*100) * w
		sumWeights += w
	}
	if sumWeights > 0 {
		avg := int(sumWeighted / sumWeights)
		rep.AvgRatingX100 = &avg
	}

	successBP := (f.Completed*10000 + priorProjects*5000) / (finished + priorProjects)
	ratingX100 := (sumWeighted + neutralRatingX100*priorReviewWeight) / (sumWeights + priorReviewWeight)
	ratingBP := int((ratingX100 - 100) * 10000 / 400) // 1 star = 0, 5 stars = 10000
	experienceBP := 5000 + min(finished, 10)*500

	rep.ScoreBP = clamp((successBP*5+ratingBP*3+experienceBP*2)/10, 0, 10000)
	return rep
}

// TrustWeight is the anti-fraud factor: how much a user's opinion counts.
// 60% comes from account age (full after 90 days) and 40% from successful
// projects (full after 5). A brand new account gets the minimum 10%, so
// creating many fake accounts to give 5-star reviews barely helps.
func TrustWeight(accountAgeDays, completed int) int {
	ageBP := min(max(accountAgeDays, 0), trustFullAgeDays) * 10000 / trustFullAgeDays
	expBP := min(max(completed, 0), trustFullCompletedAt) * 10000 / trustFullCompletedAt
	return clamp((ageBP*6+expBP*4)/10, minTrustBP, 10000)
}

func ageInDays(since, now time.Time) int {
	return int(now.Sub(since).Hours() / 24)
}

// Review is one party's rating of the other after a finished swap.
type Review struct {
	ID             string
	SwapID         string
	AuthorID       string
	AuthorUsername string
	TargetID       string
	Rating         int // 1..5 stars
	Comment        string
	CreatedAt      time.Time
}

// ReviewInput is what the author sends.
type ReviewInput struct {
	Rating  int
	Comment string
}

// Normalize trims the comment.
func (in *ReviewInput) Normalize() {
	in.Comment = strings.TrimSpace(in.Comment)
}

// Validate checks the stars and the comment length (same limits as the table).
func (in ReviewInput) Validate() error {
	if in.Rating < 1 || in.Rating > 5 {
		return &ValidationError{"rating", "must be 1-5"}
	}
	if utf8.RuneCountInString(in.Comment) > 1000 {
		return &ValidationError{"comment", "must be at most 1000 characters"}
	}
	return nil
}
