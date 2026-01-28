package model

import "time"

// Report represents the complete Entropia analysis report
// This schema matches the existing manual artifacts in /artifacts/
type Report struct {
	Subject   string    `json:"subject"`             // Subject of the report (e.g., "Laksa Origin")
	SourceURL string    `json:"source_url"`          // URL that was scanned
	FetchedAt time.Time `json:"fetched_at"`          // When the scan occurred
	FetchMeta FetchMeta `json:"fetch_meta"`          // HTTP metadata

	Claims   []Claim    `json:"claims"`              // Extracted claims
	Evidence []Evidence `json:"evidence"`            // Extracted evidence links

	Validation []ValidationResult `json:"validation,omitempty"` // Evidence validation results

	Score      Score      `json:"score"`             // Support index and scoring breakdown
	Principles Principles `json:"principles"`        // Core principles applied

	LLM *LLMSummary `json:"llm,omitempty"`          // Optional LLM summary (separate, never affects score)
}

// FetchMeta contains HTTP metadata from fetching the source
type FetchMeta struct {
	StatusCode   int               `json:"status_code"`
	ContentType  string            `json:"content_type,omitempty"`
	LastModified string            `json:"last_modified,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// Score represents the transparent scoring breakdown
type Score struct {
	Index      int      `json:"index"`       // Overall support index (0-100)
	Confidence string   `json:"confidence"`  // "low", "medium", "high"
	Conflict   bool     `json:"conflict"`    // Whether conflicting claims detected
	Signals    []Signal `json:"signals"`     // Diagnostic signals with transparent data
}

// Signal represents a diagnostic signal with transparent scoring data
type Signal struct {
	Type        SignalType             `json:"type"`                  // Signal classification
	Severity    SignalSeverity         `json:"severity"`              // info, warning, critical
	Description string                 `json:"description"`           // Human-readable description
	Data        map[string]interface{} `json:"data,omitempty"`        // Transparent scoring data (formulas, inputs)
}

// SignalType classifies the type of diagnostic signal
type SignalType string

const (
	SignalEvidenceCoverage      SignalType = "evidence_coverage"       // Claims-to-evidence ratio
	SignalAuthorityDistribution SignalType = "authority_distribution"  // Authority tier balance
	SignalFreshness             SignalType = "freshness"               // Age of sources
	SignalAccessibility         SignalType = "accessibility"           // Dead link ratio
	SignalConflict              SignalType = "conflict"                // Competing claims
	SignalStaleSources          SignalType = "stale_sources"           // Old citations
	SignalSecondarySourceBias   SignalType = "secondary_source_bias"   // Lack of primary sources
	SignalHighEntropy           SignalType = "high_entropy"            // High claim density, low support
	SignalCitationChurn         SignalType = "citation_churn"          // Frequent revisions
	SignalEditWar               SignalType = "edit_war"                // Wikipedia edit war detected
	SignalHistoricalEntity      SignalType = "historical_entity"       // Non-existent historical entity referenced
)

// SignalSeverity indicates the importance of the signal
type SignalSeverity string

const (
	SeverityInfo     SignalSeverity = "info"
	SeverityWarning  SignalSeverity = "warning"
	SeverityCritical SignalSeverity = "critical"
)

// Principles documents which core principles were applied
type Principles struct {
	NonNormative bool `json:"non_normative"` // Evaluates support, not truth
	Transparent  bool `json:"transparent"`   // All scoring explainable
	Symmetric    bool `json:"symmetric"`     // Same rules for all sources
}

// DefaultPrinciples returns the standard Entropia principles
func DefaultPrinciples() Principles {
	return Principles{
		NonNormative: true,
		Transparent:  true,
		Symmetric:    true,
	}
}

// LLMSummary contains optional LLM-generated summary
// CRITICAL: This never affects scoring and is clearly separated
type LLMSummary struct {
	Enabled        bool     `json:"enabled"`
	Provider       string   `json:"provider,omitempty"`      // openai, anthropic, ollama
	Model          string   `json:"model,omitempty"`         // Model name
	StrictEvidence bool     `json:"strict_evidence"`         // Whether citation enforcement was enabled
	SummaryMD      string   `json:"summary_md,omitempty"`    // Markdown summary
	Warnings       []string `json:"warnings,omitempty"`      // Any issues (e.g., citation leaks detected)
}

// SubjectFromURL extracts a reasonable subject name from the URL
func SubjectFromURL(rawURL string) string {
	// Simple extraction from URL path
	// This will be enhanced in the fetcher implementation
	return rawURL
}
