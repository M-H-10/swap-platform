package domain

import (
	"errors"
	"testing"
)

func TestCalculateSwapTermsEqualParties(t *testing.T) {
	newUser := CreatorStats{ReputationBP: 5000}
	terms := CalculateSwapTerms(100_000_00, 3740, newUser, newUser)

	if terms.ShareABP != 5000 || terms.ShareBBP != 5000 {
		t.Fatalf("equal users should split 50/50, got %d/%d", terms.ShareABP, terms.ShareBBP)
	}
	// 10% of 100 000 rubles = 10 000 rubles = 1 000 000 kopecks, half each.
	if terms.EscrowAmount != 1_000_000 || terms.PenaltyA != 500_000 || terms.PenaltyB != 500_000 {
		t.Fatalf("unexpected money: %+v", terms)
	}
	// gain 5 000 000 x 37.4% - loss 500 000 x 62.6% = 1 557 000
	if terms.RiskA.ExpectedValue != 1_557_000 {
		t.Fatalf("expected value = %d, want 1557000", terms.RiskA.ExpectedValue)
	}
}

func TestCalculateSwapTermsTrustedPartyGetsMore(t *testing.T) {
	trusted := CreatorStats{ReputationBP: 8000, SuccessRateBP: 9000, Completed: 9, Failed: 1}
	newcomer := CreatorStats{ReputationBP: 5000}
	terms := CalculateSwapTerms(100_000_00, 6000, trusted, newcomer)

	// 70% x 6153 (reputation) + 30% x 6428 (success rate) = 6235
	if terms.ShareABP != 6235 || terms.ShareBBP != 3765 {
		t.Fatalf("shares = %d/%d, want 6235/3765", terms.ShareABP, terms.ShareBBP)
	}
	// The newcomer has the smaller share, so it locks the larger penalty.
	if terms.PenaltyA != 376_500 || terms.PenaltyB != 623_500 {
		t.Fatalf("penalties = %d/%d, want 376500/623500", terms.PenaltyA, terms.PenaltyB)
	}
}

func TestPenaltiesAlwaysAddUpToEscrow(t *testing.T) {
	// Odd numbers on purpose: integer division must never lose a kopeck.
	for _, budget := range []int64{100_00, 123_457, 99_999_99, 1_000_000_000_00} {
		a := CreatorStats{ReputationBP: 7333}
		b := CreatorStats{ReputationBP: 2111}
		terms := CalculateSwapTerms(budget, 5000, a, b)
		if terms.PenaltyA+terms.PenaltyB != terms.EscrowAmount {
			t.Fatalf("budget %d: %d + %d != %d", budget, terms.PenaltyA, terms.PenaltyB, terms.EscrowAmount)
		}
		if terms.ShareABP+terms.ShareBBP != 10000 {
			t.Fatalf("budget %d: shares do not add up to 100%%", budget)
		}
	}
}

func TestSharesStayBetween5And95Percent(t *testing.T) {
	strong := CreatorStats{ReputationBP: 10000, SuccessRateBP: 10000, Completed: 50}
	weak := CreatorStats{ReputationBP: 0, SuccessRateBP: 0, Failed: 50}
	terms := CalculateSwapTerms(100_000_00, 5000, strong, weak)
	if terms.ShareABP != 9500 || terms.ShareBBP != 500 {
		t.Fatalf("shares = %d/%d, want 9500/500", terms.ShareABP, terms.ShareBBP)
	}
}

func TestFee(t *testing.T) {
	if f := Fee(500_000); f != 2_500 { // 0.5% of 5000 rubles = 25 rubles
		t.Fatalf("fee = %d, want 2500", f)
	}
}

func TestParseOutcome(t *testing.T) {
	if o, err := ParseOutcome(" Success "); err != nil || o != OutcomeSuccess {
		t.Fatalf("got %q, %v", o, err)
	}
	var ve *ValidationError
	if _, err := ParseOutcome("maybe"); !errors.As(err, &ve) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
