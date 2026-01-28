package score

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ppiankov/entropia/internal/model"
)

// Scorer calculates the support index and generates signals
type Scorer struct{}

// NewScorer creates a new scorer
func NewScorer() *Scorer {
	return &Scorer{}
}

// Calculate calculates the support score and generates diagnostic signals
func (s *Scorer) Calculate(claims []model.Claim, evidence []model.Evidence, validation []model.ValidationResult) model.Score {
	var signals []model.Signal

	// 1. Evidence Coverage (0-40 points)
	coverageScore, coverageSignal := s.calculateCoverage(claims, evidence)
	signals = append(signals, coverageSignal)

	// 2. Authority Distribution (0-30 points)
	authorityScore, authoritySignal := s.calculateAuthority(validation)
	signals = append(signals, authoritySignal)

	// 3. Freshness (0-20 points)
	freshnessScore, freshnessSignal := s.calculateFreshness(validation)
	signals = append(signals, freshnessSignal)

	// 4. Accessibility (0-10 points)
	accessScore, accessSignal := s.calculateAccessibility(validation)
	signals = append(signals, accessSignal)

	// 5. Conflict Detection (penalty)
	conflictDetected, conflictSignal := s.detectConflict(claims)
	if conflictDetected {
		signals = append(signals, conflictSignal)
	}

	// 6. Freshness Anomaly Detection
	freshnessAnomalySignal := s.detectFreshnessAnomaly(validation, len(evidence))
	if freshnessAnomalySignal.Type != "" {
		signals = append(signals, freshnessAnomalySignal)
	}

	// Calculate total score
	totalScore := coverageScore + authorityScore + freshnessScore + accessScore

	// Apply conflict penalty
	if conflictDetected {
		totalScore -= 10
		if totalScore < 0 {
			totalScore = 0
		}
	}

	// Determine confidence level
	confidence := s.determineConfidence(totalScore, len(evidence), conflictDetected)

	return model.Score{
		Index:      totalScore,
		Confidence: confidence,
		Conflict:   conflictDetected,
		Signals:    signals,
	}
}

// calculateCoverage calculates evidence coverage score (0-40 points)
func (s *Scorer) calculateCoverage(claims []model.Claim, evidence []model.Evidence) (int, model.Signal) {
	claimCount := len(claims)
	evidenceCount := len(evidence)

	if claimCount == 0 {
		return 0, model.Signal{
			Type:        model.SignalEvidenceCoverage,
			Severity:    model.SeverityCritical,
			Description: "No claims extracted",
			Data: map[string]interface{}{
				"claims":   0,
				"evidence": evidenceCount,
			},
		}
	}

	ratio := float64(evidenceCount) / float64(claimCount)
	score := int(math.Min(ratio*40, 40))

	severity := model.SeverityInfo
	if ratio < 0.5 {
		severity = model.SeverityCritical
	} else if ratio < 1.0 {
		severity = model.SeverityWarning
	}

	return score, model.Signal{
		Type:        model.SignalEvidenceCoverage,
		Severity:    severity,
		Description: fmt.Sprintf("Evidence-to-claim ratio: %.2f", ratio),
		Data: map[string]interface{}{
			"claims":   claimCount,
			"evidence": evidenceCount,
			"ratio":    ratio,
			"score":    score,
			"formula":  "min(evidence_count / claim_count * 40, 40)",
		},
	}
}

// calculateAuthority calculates authority distribution score (0-30 points)
func (s *Scorer) calculateAuthority(validation []model.ValidationResult) (int, model.Signal) {
	if len(validation) == 0 {
		return 0, model.Signal{
			Type:        model.SignalAuthorityDistribution,
			Severity:    model.SeverityWarning,
			Description: "No validation data available",
			Data:        map[string]interface{}{"validated": 0},
		}
	}

	primaryCount := 0
	secondaryCount := 0
	tertiaryCount := 0

	for _, v := range validation {
		switch v.Authority {
		case model.TierPrimary:
			primaryCount++
		case model.TierSecondary:
			secondaryCount++
		case model.TierTertiary:
			tertiaryCount++
		}
	}

	total := len(validation)
	weightedSum := float64(primaryCount*3 + secondaryCount*2 + tertiaryCount*1)
	maxPossible := float64(total * 3)
	score := int((weightedSum / maxPossible) * 30)

	severity := model.SeverityInfo
	if primaryCount == 0 {
		severity = model.SeverityWarning
	}

	return score, model.Signal{
		Type:        model.SignalAuthorityDistribution,
		Severity:    severity,
		Description: fmt.Sprintf("Authority distribution: %d primary, %d secondary, %d tertiary", primaryCount, secondaryCount, tertiaryCount),
		Data: map[string]interface{}{
			"primary":   primaryCount,
			"secondary": secondaryCount,
			"tertiary":  tertiaryCount,
			"total":     total,
			"score":     score,
			"formula":   "(primary*3 + secondary*2 + tertiary*1) / (total*3) * 30",
		},
	}
}

// calculateFreshness calculates freshness score (0-20 points)
func (s *Scorer) calculateFreshness(validation []model.ValidationResult) (int, model.Signal) {
	var ages []int
	for _, v := range validation {
		if v.Age != nil {
			ages = append(ages, *v.Age)
		}
	}

	if len(ages) == 0 {
		return 10, model.Signal{
			Type:        model.SignalFreshness,
			Severity:    model.SeverityInfo,
			Description: "No freshness data available (assuming moderate)",
			Data:        map[string]interface{}{"samples": 0, "score": 10},
		}
	}

	// Calculate median age
	sort.Ints(ages)
	medianAge := ages[len(ages)/2]
	medianAgeYears := float64(medianAge) / 365.0

	// Score: 20 points for fresh, decreasing by 5 points per year
	score := 20 - int(medianAgeYears*5)
	if score < 0 {
		score = 0
	}

	severity := model.SeverityInfo
	if medianAgeYears > 3 {
		severity = model.SeverityCritical
	} else if medianAgeYears > 1 {
		severity = model.SeverityWarning
	}

	// Calculate percentage of sources with freshness data
	totalSources := len(validation)
	freshnessPercentage := float64(len(ages)) / float64(totalSources) * 100

	description := fmt.Sprintf("Median age: %.1f years", medianAgeYears)
	if freshnessPercentage < 50 {
		description = fmt.Sprintf("Median age: %.1f years (%d/%d sources with Last-Modified)",
			medianAgeYears, len(ages), totalSources)
	}

	return score, model.Signal{
		Type:        model.SignalFreshness,
		Severity:    severity,
		Description: description,
		Data: map[string]interface{}{
			"median_age_days":       medianAge,
			"median_age_years":      medianAgeYears,
			"samples":               len(ages),
			"total_sources":         totalSources,
			"freshness_coverage":    freshnessPercentage,
			"score":                 score,
			"formula":               "20 - min(median_age_years * 5, 20)",
		},
	}
}

// calculateAccessibility calculates accessibility score (0-10 points)
func (s *Scorer) calculateAccessibility(validation []model.ValidationResult) (int, model.Signal) {
	if len(validation) == 0 {
		return 0, model.Signal{
			Type:        model.SignalAccessibility,
			Severity:    model.SeverityWarning,
			Description: "No validation data available",
			Data:        map[string]interface{}{"validated": 0},
		}
	}

	accessibleCount := 0
	for _, v := range validation {
		if v.IsAccessible {
			accessibleCount++
		}
	}

	ratio := float64(accessibleCount) / float64(len(validation))
	score := int(ratio * 10)

	severity := model.SeverityInfo
	if ratio < 0.5 {
		severity = model.SeverityCritical
	} else if ratio < 0.8 {
		severity = model.SeverityWarning
	}

	return score, model.Signal{
		Type:        model.SignalAccessibility,
		Severity:    severity,
		Description: fmt.Sprintf("Accessibility: %d/%d (%.0f%%)", accessibleCount, len(validation), ratio*100),
		Data: map[string]interface{}{
			"accessible": accessibleCount,
			"total":      len(validation),
			"ratio":      ratio,
			"score":      score,
			"formula":    "(accessible_count / total) * 10",
		},
	}
}

// detectConflict detects conflicting claims
func (s *Scorer) detectConflict(claims []model.Claim) (bool, model.Signal) {
	// Look for origin-related claims with different countries/entities
	originClaims := []string{}
	countries := make(map[string]bool)

	for _, claim := range claims {
		lower := strings.ToLower(claim.Text)
		if strings.Contains(lower, "origin") || strings.Contains(lower, "originated") {
			originClaims = append(originClaims, claim.Text)

			// Extract potential country names
			for _, country := range []string{"malaysia", "indonesia", "england", "wales", "uk", "britain", "china", "india", "thailand"} {
				if strings.Contains(lower, country) {
					countries[country] = true
				}
			}
		}
	}

	conflictDetected := len(countries) >= 2 && len(originClaims) >= 2

	if conflictDetected {
		return true, model.Signal{
			Type:        model.SignalConflict,
			Severity:    model.SeverityWarning,
			Description: fmt.Sprintf("Conflicting origin claims detected (%d different entities)", len(countries)),
			Data: map[string]interface{}{
				"origin_claims": len(originClaims),
				"entities":      len(countries),
				"penalty":       10,
			},
		}
	}

	return false, model.Signal{}
}

// detectFreshnessAnomaly detects when sources are suspiciously recent for a topic
// This can indicate ongoing content disputes or constant editing wars
func (s *Scorer) detectFreshnessAnomaly(validation []model.ValidationResult, totalEvidence int) model.Signal {
	// Only check if we have substantial evidence with freshness data
	var ages []int
	for _, v := range validation {
		if v.Age != nil {
			ages = append(ages, *v.Age)
		}
	}

	// Need at least 20 sources with freshness data to make this assessment
	if len(ages) < 20 {
		return model.Signal{} // Return empty signal
	}

	// Calculate median age
	sort.Ints(ages)
	medianAge := ages[len(ages)/2]
	medianAgeYears := float64(medianAge) / 365.0

	// Anomaly: Many sources but all very recent (<1 year)
	// This suggests ongoing disputes or frequent content changes
	if medianAgeYears < 1.0 && len(ages) > 50 {
		return model.Signal{
			Type:        model.SignalFreshnessAnomaly,
			Severity:    model.SeverityWarning,
			Description: "Suspiciously recent sources: all evidence very new despite topic likely being historical",
			Data: map[string]interface{}{
				"median_age_years": medianAgeYears,
				"sources_with_age": len(ages),
				"total_evidence":   totalEvidence,
				"explanation":      "For topics with historical significance, having all sources be very recent (<1 year) suggests ongoing content disputes, frequent revisions, or edit wars rather than stable, established information",
			},
		}
	}

	return model.Signal{} // No anomaly detected
}

// determineConfidence determines the confidence level based on the score
func (s *Scorer) determineConfidence(score int, evidenceCount int, conflict bool) string {
	if conflict {
		return "low-medium"
	}

	if evidenceCount < 3 {
		return "low"
	}

	if score >= 80 {
		return "high"
	} else if score >= 60 {
		return "medium"
	} else {
		return "low"
	}
}
