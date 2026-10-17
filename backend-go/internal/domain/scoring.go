package domain

import "time"

// The scoring formula from the specification (section 11.2):
//
//	Score = reputation + success + experience - complexity - risks + budget + deadline
//
// Everything is an integer in basis points (bp): 10000 bp = 100 points.
// The score starts at 5000 (a neutral 50 out of 100), every factor adds
// or removes points, and the result is kept between 0 and 10000. It is
// then mapped to a success probability between 5% and 95%: the formula
// never claims that a project is certain to succeed or to fail.
//
// The same input always gives the same output (no randomness, no clock
// inside), so the result can be explained and tested line by line.

const (
	scoreBase        = 5000
	scoreMax         = 10000
	probabilityMinBP = 500  // 5%
	probabilityMaxBP = 9500 // 95%
)

// CreatorStats is what the formula needs to know about the project's creator.
type CreatorStats struct {
	ReputationBP  int // latest reputation, 0..10000
	SuccessRateBP int // share of finished projects that succeeded
	Completed     int // projects that ended in success
	Failed        int // projects that ended in failure
	AccountSince  time.Time
}

// ScoreInput is everything the formula looks at.
type ScoreInput struct {
	Budget         int64 // kopecks
	DurationDays   int
	Complexity     int
	TeamSize       int
	ReputationBP   int
	SuccessRateBP  int
	Completed      int
	Failed         int
	AccountAgeDays int
}

// ScoreFactor is one line of the explanation: which factor, how many points.
type ScoreFactor struct {
	Name   string
	Points int
}

// ScoreResult is the final score, the probability, and the explanation.
type ScoreResult struct {
	ScoreBP              int
	SuccessProbabilityBP int
	Factors              []ScoreFactor
}

// NewScoreInput combines the project and its creator into one input.
// The current time is passed in, so the formula itself never reads a clock.
func NewScoreInput(p ProjectInput, c CreatorStats, now time.Time) ScoreInput {
	return ScoreInput{
		Budget:         p.Budget,
		DurationDays:   p.DurationDays,
		Complexity:     p.Complexity,
		TeamSize:       p.TeamSize,
		ReputationBP:   c.ReputationBP,
		SuccessRateBP:  c.SuccessRateBP,
		Completed:      c.Completed,
		Failed:         c.Failed,
		AccountAgeDays: int(now.Sub(c.AccountSince).Hours() / 24),
	}
}

// CalculateScore runs the formula.
func CalculateScore(in ScoreInput) ScoreResult {
	factors := []ScoreFactor{
		{"reputation", reputationPoints(in.ReputationBP)},
		{"success_rate", successRatePoints(in.SuccessRateBP, in.Completed+in.Failed)},
		{"experience", experiencePoints(in.Completed)},
		{"complexity", complexityPoints(in.Complexity)},
		{"account_age", accountAgePoints(in.AccountAgeDays)},
		{"team_size", teamSizePoints(in.TeamSize)},
		{"budget", budgetPoints(in.Budget, in.TeamSize, in.DurationDays)},
		{"deadline", deadlinePoints(in.DurationDays, in.Complexity)},
	}

	score := scoreBase
	for _, f := range factors {
		score += f.Points
	}
	score = clamp(score, 0, scoreMax)

	return ScoreResult{
		ScoreBP:              score,
		SuccessProbabilityBP: probabilityMinBP + score*(probabilityMaxBP-probabilityMinBP)/scoreMax,
		Factors:              factors,
	}
}

// reputation: up to +-1500. Average reputation (5000) gives 0.
func reputationPoints(reputationBP int) int {
	return (clamp(reputationBP, 0, scoreMax) - 5000) * 3 / 10
}

// success: up to +-1000, but only when the creator has finished projects.
// With no history there is nothing to judge, so the factor stays 0.
func successRatePoints(successRateBP, finished int) int {
	if finished == 0 {
		return 0
	}
	return (clamp(successRateBP, 0, scoreMax) - 5000) / 5
}

// experience: +50 for every successful project, at most +1000 (20 projects).
func experiencePoints(completed int) int {
	return clamp(completed, 0, 20) * 50
}

// complexity: 5 is average (0). Easier projects gain, harder ones lose:
// complexity 1 gives +1000, complexity 10 gives -1250.
func complexityPoints(complexity int) int {
	return -(clamp(complexity, 1, 10) - 5) * 250
}

// risk 1, account age (anti-fraud): brand new accounts count for less.
func accountAgePoints(days int) int {
	switch {
	case days < 7:
		return -1000
	case days < 30:
		return -500
	default:
		return 0
	}
}

// risk 2, team size: every member above 10 makes coordination harder,
// -25 each, at most -500.
func teamSizePoints(teamSize int) int {
	if teamSize <= 10 {
		return 0
	}
	return -(clamp(teamSize, 10, 30) - 10) * 25
}

// budget: money per person per day. Too little money is a classic reason
// for failure. Below 500 rubles: -1000; 500 to 2000 rubles: 0; 2000+: +500.
func budgetPoints(budget int64, teamSize, durationDays int) int {
	personDays := int64(max(teamSize, 1)) * int64(max(durationDays, 1))
	perPersonDay := budget / personDays // kopecks
	switch {
	case perPersonDay < 500_00:
		return -1000
	case perPersonDay < 2000_00:
		return 0
	default:
		return 500
	}
}

// deadline: a rough need of one week per complexity point. Half of that
// or less is a rush (-1000), less than that is tight (-400), more than a
// year drags on (-300), anything else is realistic (+300).
func deadlinePoints(durationDays, complexity int) int {
	needed := clamp(complexity, 1, 10) * 7
	switch {
	case durationDays*2 <= needed:
		return -1000
	case durationDays < needed:
		return -400
	case durationDays > 365:
		return -300
	default:
		return 300
	}
}

func clamp(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
