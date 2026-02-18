package llm

import (
	"context"
	"fmt"

	"github.com/ppiankov/entropia/internal/model"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// Summarize generates a summary of the report with strict evidence mode
	Summarize(ctx context.Context, req SummarizeRequest) (*SummarizeResponse, error)

	// IsAvailable checks if the provider is properly configured and accessible
	IsAvailable(ctx context.Context) bool
}

// SummarizeRequest contains the input for LLM summarization
type SummarizeRequest struct {
	// Report is the Entropia scan report to summarize
	Report model.Report

	// EvidenceURLs is the STRICT allowlist of URLs the LLM can cite
	// This prevents hallucination - LLM cannot reference any URL not in this list
	EvidenceURLs []string

	// Prompt is an optional custom prompt (if empty, use default)
	Prompt string

	// Model is the specific model to use (provider-specific)
	Model string

	// MaxTokens limits the response length
	MaxTokens int
}

// SummarizeResponse contains the LLM's summary output
type SummarizeResponse struct {
	// Summary is the generated summary text
	Summary string

	// CitedURLs are the URLs the LLM actually cited (for verification)
	CitedURLs []string

	// Model is the model that generated the response
	Model string

	// TokensUsed tracks token consumption
	TokensUsed int
}

// Config holds LLM provider configuration
type Config struct {
	// Provider name: "openai", "anthropic", "ollama", ""
	Provider string

	// Model name (provider-specific)
	Model string

	// APIKey for OpenAI/Anthropic
	APIKey string

	// BaseURL for custom endpoints (e.g., Ollama)
	BaseURL string

	// Timeout for API requests
	Timeout int // seconds

	// StrictEvidence enforces URL allowlist (should always be true)
	StrictEvidence bool

	// MaxTokens for response generation
	MaxTokens int

	// Proxy settings
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Provider:       "", // Disabled by default
		Model:          "",
		Timeout:        30,
		StrictEvidence: true, // CRITICAL: Always enforce
		MaxTokens:      1000,
	}
}

// BuildPrompt constructs the default prompt for summarization with strict evidence mode
func BuildPrompt(report model.Report, evidenceURLs []string) string {
	prompt := fmt.Sprintf(`You are summarizing an Entropia report. Entropia evaluates how well claims are supported by evidence - it NEVER asserts truth or correctness.

CRITICAL RULES:
1. You MUST ONLY cite URLs from this allowed list:
%s

2. DO NOT infer, speculate, or cite external sources beyond this list.
3. If evidence is insufficient or missing, state that explicitly.
4. Focus on SUPPORT QUALITY, not truth. Use phrases like:
   - "The claim is supported by X sources..."
   - "Evidence is lacking for..."
   - "Cited sources date from..."
5. Never say "this is true" or "this is false" - only describe evidence.

Report Summary:
- Subject: %s
- Support Index: %d/100
- Claims Identified: %d
- Evidence Links: %d
- Validated Links: %d accessible, %d dead/inaccessible

Key Signals:
`, joinURLs(evidenceURLs), report.Subject, report.Score.Index, len(report.Claims), len(report.Evidence), countAccessible(report.Validation), countDead(report.Validation))

	// Add top 3 signals
	for i, signal := range report.Score.Signals {
		if i >= 3 {
			break
		}
		prompt += fmt.Sprintf("- %s: %s\n", signal.Type, signal.Description)
	}

	prompt += "\nProvide a 3-4 sentence summary focusing on evidence quality, not truth."

	return prompt
}

// Helper functions

func joinURLs(urls []string) string {
	if len(urls) == 0 {
		return "(No evidence URLs available)"
	}
	result := ""
	for i, url := range urls {
		if i >= 20 { // Limit to first 20 to avoid token bloat
			result += fmt.Sprintf("\n... and %d more URLs", len(urls)-20)
			break
		}
		result += fmt.Sprintf("\n- %s", url)
	}
	return result
}

func countAccessible(validation []model.ValidationResult) int {
	count := 0
	for _, v := range validation {
		if v.IsAccessible {
			count++
		}
	}
	return count
}

func countDead(validation []model.ValidationResult) int {
	count := 0
	for _, v := range validation {
		if !v.IsAccessible {
			count++
		}
	}
	return count
}
