package score

import (
	"testing"

	"github.com/ppiankov/entropia/internal/model"
)

func TestScorer_Calculate_BasicScoring(t *testing.T) {
	scorer := NewScorer()

	// Test case: 10 claims, 5 evidence links, all accessible, all tertiary, no conflict
	claims := make([]model.Claim, 10)
	for i := 0; i < 10; i++ {
		claims[i] = model.Claim{
			Text:      "Test claim",
			Heuristic: "test",
			Sentence:  i,
		}
	}

	evidence := make([]model.Evidence, 5)
	for i := 0; i < 5; i++ {
		evidence[i] = model.Evidence{
			URL:        "https://example.com",
			Kind:       model.EvidenceKindExternalLink,
			Host:       "example.com",
			IsSameHost: false,
			Authority:  model.TierTertiary,
		}
	}

	validation := make([]model.ValidationResult, 5)
	for i := 0; i < 5; i++ {
		validation[i] = model.ValidationResult{
			URL:          "https://example.com",
			IsAccessible: true,
			StatusCode:   200,
			Authority:    model.TierTertiary,
			IsDead:       false,
			IsStale:      false,
		}
	}

	result := scorer.Calculate(claims, evidence, validation)

	// Check that index is between 0 and 100
	if result.Index < 0 || result.Index > 100 {
		t.Errorf("Expected index between 0 and 100, got %d", result.Index)
	}

	// Coverage should be 20 points (5/10 * 40 = 20)
	// Authority should be low (all tertiary)
	// Freshness we can't test without dates
	// Accessibility should be 10 points (100% accessible)
	// Total should be around 30-40 points depending on freshness

	if result.Index < 20 {
		t.Errorf("Expected index >= 20 for this test case, got %d", result.Index)
	}

	// Check confidence is set
	if result.Confidence == "" {
		t.Error("Expected confidence to be set")
	}

	// Check signals exist
	if len(result.Signals) == 0 {
		t.Error("Expected at least one signal")
	}
}

func TestScorer_Calculate_EmptyClaims(t *testing.T) {
	scorer := NewScorer()

	claims := []model.Claim{}
	evidence := []model.Evidence{}
	validation := []model.ValidationResult{}

	result := scorer.Calculate(claims, evidence, validation)

	// Should not panic and should return valid result
	if result.Index < 0 || result.Index > 100 {
		t.Errorf("Expected index between 0 and 100 for empty input, got %d", result.Index)
	}

	if result.Confidence == "" {
		t.Error("Expected confidence to be set even for empty input")
	}
}

func TestScorer_Calculate_HighQuality(t *testing.T) {
	scorer := NewScorer()

	// Test case: 10 claims, 15 evidence links (over-evidenced), all accessible, all primary, no conflict
	claims := make([]model.Claim, 10)
	for i := 0; i < 10; i++ {
		claims[i] = model.Claim{
			Text:      "Test claim",
			Heuristic: "test",
			Sentence:  i,
		}
	}

	evidence := make([]model.Evidence, 15)
	for i := 0; i < 15; i++ {
		evidence[i] = model.Evidence{
			URL:        "https://doi.org/example",
			Kind:       model.EvidenceKindCitation,
			Host:       "doi.org",
			IsSameHost: false,
			Authority:  model.TierPrimary,
		}
	}

	validation := make([]model.ValidationResult, 15)
	for i := 0; i < 15; i++ {
		validation[i] = model.ValidationResult{
			URL:          "https://doi.org/example",
			IsAccessible: true,
			StatusCode:   200,
			Authority:    model.TierPrimary,
			IsDead:       false,
			IsStale:      false,
		}
	}

	result := scorer.Calculate(claims, evidence, validation)

	// Coverage should be 40 points (capped at 150% = 40 max)
	// Authority should be 30 points (all primary)
	// Accessibility should be 10 points (100% accessible)
	// Freshness depends on dates
	// Total should be high (80+ if fresh sources)

	if result.Index < 70 {
		t.Errorf("Expected high index (>=70) for high-quality evidence, got %d", result.Index)
	}

	if result.Confidence != "high" {
		t.Errorf("Expected high confidence for high-quality evidence, got %s", result.Confidence)
	}
}

func TestScorer_Calculate_DeadLinks(t *testing.T) {
	scorer := NewScorer()

	claims := make([]model.Claim, 10)
	for i := 0; i < 10; i++ {
		claims[i] = model.Claim{
			Text:      "Test claim",
			Heuristic: "test",
			Sentence:  i,
		}
	}

	evidence := make([]model.Evidence, 10)
	for i := 0; i < 10; i++ {
		evidence[i] = model.Evidence{
			URL:        "https://example.com",
			Kind:       model.EvidenceKindExternalLink,
			Host:       "example.com",
			IsSameHost: false,
			Authority:  model.TierSecondary,
		}
	}

	// Half the links are dead
	validation := make([]model.ValidationResult, 10)
	for i := 0; i < 10; i++ {
		validation[i] = model.ValidationResult{
			URL:          "https://example.com",
			IsAccessible: i%2 == 0, // 50% accessible
			StatusCode:   map[bool]int{true: 200, false: 404}[i%2 == 0],
			Authority:    model.TierSecondary,
			IsDead:       i%2 != 0, // 50% dead
			IsStale:      false,
		}
	}

	result := scorer.Calculate(claims, evidence, validation)

	// Accessibility should be 5 points (50% accessible = 5/10)
	// This should significantly lower the score

	// Check that dead links signal exists
	hasDeadLinkSignal := false
	for _, signal := range result.Signals {
		if signal.Type == model.SignalAccessibility {
			hasDeadLinkSignal = true
			break
		}
	}

	if !hasDeadLinkSignal {
		t.Error("Expected accessibility signal for dead links")
	}
}
