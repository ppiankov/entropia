package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/ppiankov/entropia/internal/model"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	name      string
	available bool
	response  *SummarizeResponse
	err       error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Summarize(ctx context.Context, req SummarizeRequest) (*SummarizeResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockProvider) IsAvailable(ctx context.Context) bool {
	return m.available
}

func TestNewSummarizer_DisabledProvider(t *testing.T) {
	config := Config{
		Provider: "", // Empty = disabled
	}

	summarizer, err := NewSummarizer(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summarizer.provider != nil {
		t.Error("Expected provider to be nil when disabled")
	}

	if summarizer.IsEnabled() {
		t.Error("Expected summarizer to be disabled")
	}

	if summarizer.ProviderName() != "" {
		t.Error("Expected empty provider name when disabled")
	}
}

func TestSummarizer_GenerateSummary_Disabled(t *testing.T) {
	// Create summarizer with nil provider (disabled)
	summarizer := &Summarizer{
		provider: nil,
		config:   Config{},
	}

	report := model.Report{
		Subject: "Test Subject",
	}

	summary, err := summarizer.GenerateSummary(context.Background(), report)

	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}

	if summary != nil {
		t.Error("Expected nil summary when provider disabled")
	}
}

func TestSummarizer_GenerateSummary_ProviderUnavailable(t *testing.T) {
	mockProvider := &MockProvider{
		name:      "test-provider",
		available: false, // Provider not available
	}

	summarizer := &Summarizer{
		provider: mockProvider,
		config:   Config{StrictEvidence: true},
	}

	report := model.Report{
		Subject: "Test Subject",
	}

	summary, err := summarizer.GenerateSummary(context.Background(), report)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary object with warnings")
	}

	if summary.Enabled {
		t.Error("Expected summary to be marked as disabled")
	}

	if len(summary.Warnings) == 0 {
		t.Error("Expected warning about provider unavailability")
	}

	// Check warning message
	found := false
	for _, warning := range summary.Warnings {
		if strings.Contains(warning, "not available") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning to mention provider unavailability")
	}
}

func TestSummarizer_GenerateSummary_Success(t *testing.T) {
	mockProvider := &MockProvider{
		name:      "test-provider",
		available: true,
		response: &SummarizeResponse{
			Summary:    "This is a test summary.",
			CitedURLs:  []string{"https://example.com/1", "https://example.com/2"},
			Model:      "test-model",
			TokensUsed: 150,
		},
	}

	summarizer := &Summarizer{
		provider: mockProvider,
		config: Config{
			Model:          "test-model",
			StrictEvidence: true,
		},
	}

	report := model.Report{
		Subject: "Test Subject",
		Evidence: []model.Evidence{
			{URL: "https://example.com/1"},
			{URL: "https://example.com/2"},
		},
	}

	summary, err := summarizer.GenerateSummary(context.Background(), report)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary to be generated")
	}

	if !summary.Enabled {
		t.Error("Expected summary to be enabled")
	}

	if summary.Provider != "test-provider" {
		t.Errorf("Expected provider 'test-provider', got '%s'", summary.Provider)
	}

	if summary.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", summary.Model)
	}

	if !summary.StrictEvidence {
		t.Error("Expected strict evidence mode to be enabled")
	}

	if summary.SummaryMD != "This is a test summary." {
		t.Errorf("Expected summary text to match, got '%s'", summary.SummaryMD)
	}

	// Check warnings include token usage and citation verification
	foundTokens := false
	foundCitations := false
	for _, warning := range summary.Warnings {
		if strings.Contains(warning, "Tokens used") {
			foundTokens = true
		}
		if strings.Contains(warning, "Verified") && strings.Contains(warning, "citations") {
			foundCitations = true
		}
	}

	if !foundTokens {
		t.Error("Expected warning about tokens used")
	}

	if !foundCitations {
		t.Error("Expected warning about verified citations")
	}
}

func TestSummarizer_GenerateSummary_ProviderError(t *testing.T) {
	mockProvider := &MockProvider{
		name:      "test-provider",
		available: true,
		err:       &mockError{msg: "API rate limit exceeded"},
	}

	summarizer := &Summarizer{
		provider: mockProvider,
		config: Config{
			Model:          "test-model",
			StrictEvidence: true,
		},
	}

	report := model.Report{
		Subject: "Test Subject",
	}

	summary, err := summarizer.GenerateSummary(context.Background(), report)

	// Should not fail the entire scan, just return summary with warnings
	if err != nil {
		t.Errorf("Expected no error (graceful degradation), got %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary with error warning")
	}

	if !summary.Enabled {
		t.Error("Expected summary to be marked as enabled (but failed)")
	}

	if len(summary.Warnings) == 0 {
		t.Fatal("Expected warning about generation failure")
	}

	// Check warning mentions the error
	found := false
	for _, warning := range summary.Warnings {
		if strings.Contains(warning, "failed") && strings.Contains(warning, "rate limit") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning to mention error: %v", summary.Warnings)
	}
}

func TestRenderSeparateMarkdown_Disabled(t *testing.T) {
	summary := &model.LLMSummary{
		Enabled: false,
	}

	md := RenderSeparateMarkdown(summary)

	if md != "" {
		t.Error("Expected empty markdown when disabled")
	}
}

func TestRenderSeparateMarkdown_Nil(t *testing.T) {
	md := RenderSeparateMarkdown(nil)

	if md != "" {
		t.Error("Expected empty markdown when nil")
	}
}

func TestRenderSeparateMarkdown_Success(t *testing.T) {
	summary := &model.LLMSummary{
		Enabled:        true,
		Provider:       "openai",
		Model:          "gpt-4o-mini",
		StrictEvidence: true,
		SummaryMD:      "This is the generated summary content.",
		Warnings: []string{
			"Tokens used: 150",
			"Verified 5 citations",
		},
	}

	md := RenderSeparateMarkdown(summary)

	if md == "" {
		t.Fatal("Expected markdown to be generated")
	}

	// Check required sections
	requiredSections := []string{
		"# LLM Summary",
		"GENERATED CONTENT",
		"Provider",
		"openai",
		"Model",
		"gpt-4o-mini",
		"Strict Evidence Mode",
		"true",
		"This is the generated summary content.",
		"## Notes",
		"Tokens used: 150",
		"Verified 5 citations",
	}

	for _, section := range requiredSections {
		if !strings.Contains(md, section) {
			t.Errorf("Expected markdown to contain '%s'", section)
		}
	}

	// Check disclaimer is present
	if !strings.Contains(md, "determined independently") {
		t.Error("Expected disclaimer about independence from LLM")
	}
}

func TestRenderSeparateMarkdown_NoSummary(t *testing.T) {
	summary := &model.LLMSummary{
		Enabled:        true,
		Provider:       "test-provider",
		StrictEvidence: true,
		SummaryMD:      "", // Empty summary
	}

	md := RenderSeparateMarkdown(summary)

	if !strings.Contains(md, "No summary generated") {
		t.Error("Expected message about no summary")
	}
}

func TestBuildPrompt_BasicStructure(t *testing.T) {
	report := model.Report{
		Subject: "Test Article",
		Score: model.Score{
			Index: 75,
			Signals: []model.Signal{
				{Type: "evidence_coverage", Description: "Good evidence-to-claim ratio"},
				{Type: "freshness", Description: "Sources are recent"},
			},
		},
		Claims: []model.Claim{
			{Text: "Claim 1"},
			{Text: "Claim 2"},
			{Text: "Claim 3"},
		},
		Evidence: []model.Evidence{
			{URL: "https://example.com/1"},
			{URL: "https://example.com/2"},
		},
		Validation: []model.ValidationResult{
			{IsAccessible: true},
			{IsAccessible: false},
		},
	}

	evidenceURLs := []string{
		"https://example.com/1",
		"https://example.com/2",
	}

	prompt := BuildPrompt(report, evidenceURLs)

	// Check required elements
	requiredElements := []string{
		"CRITICAL RULES",
		"MUST ONLY cite URLs from this allowed list",
		"https://example.com/1",
		"https://example.com/2",
		"DO NOT infer, speculate",
		"Subject: Test Article",
		"Support Index: 75/100",
		"Claims Identified: 3",
		"Evidence Links: 2",
		"1 accessible",
		"1 dead/inaccessible",
		"evidence_coverage",
		"freshness",
		"SUPPORT QUALITY, not truth",
	}

	for _, element := range requiredElements {
		if !strings.Contains(prompt, element) {
			t.Errorf("Expected prompt to contain '%s'", element)
		}
	}
}

func TestBuildPrompt_NoEvidence(t *testing.T) {
	report := model.Report{
		Subject: "Test Article",
		Score: model.Score{
			Index:   20,
			Signals: []model.Signal{},
		},
		Claims:     []model.Claim{},
		Evidence:   []model.Evidence{},
		Validation: []model.ValidationResult{},
	}

	prompt := BuildPrompt(report, []string{})

	if !strings.Contains(prompt, "No evidence URLs available") {
		t.Error("Expected message about no evidence URLs")
	}
}

func TestBuildPrompt_ManyURLs(t *testing.T) {
	// Create 25 URLs
	evidenceURLs := make([]string, 25)
	for i := 0; i < 25; i++ {
		evidenceURLs[i] = "https://example.com/" + string(rune('a'+i))
	}

	report := model.Report{
		Subject: "Test",
		Score:   model.Score{Index: 50},
	}

	prompt := BuildPrompt(report, evidenceURLs)

	// Should limit to 20 URLs and show "... and X more"
	if !strings.Contains(prompt, "and 5 more URLs") {
		t.Error("Expected truncation message for many URLs")
	}

	// First URL should be present
	if !strings.Contains(prompt, evidenceURLs[0]) {
		t.Error("Expected first URL to be in prompt")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Provider != "" {
		t.Errorf("Expected provider to be empty (disabled), got '%s'", config.Provider)
	}

	if !config.StrictEvidence {
		t.Error("Expected strict evidence to be enabled by default (CRITICAL)")
	}

	if config.Timeout <= 0 {
		t.Error("Expected positive timeout")
	}

	if config.MaxTokens <= 0 {
		t.Error("Expected positive max tokens")
	}
}

func TestSummarizer_IsEnabled(t *testing.T) {
	// Disabled summarizer
	disabled := &Summarizer{
		provider: nil,
	}

	if disabled.IsEnabled() {
		t.Error("Expected IsEnabled() to return false when provider is nil")
	}

	// Enabled summarizer
	enabled := &Summarizer{
		provider: &MockProvider{name: "test"},
	}

	if !enabled.IsEnabled() {
		t.Error("Expected IsEnabled() to return true when provider exists")
	}
}

func TestSummarizer_ProviderName(t *testing.T) {
	// Disabled summarizer
	disabled := &Summarizer{
		provider: nil,
	}

	if disabled.ProviderName() != "" {
		t.Error("Expected empty provider name when disabled")
	}

	// Enabled summarizer
	enabled := &Summarizer{
		provider: &MockProvider{name: "test-provider"},
	}

	if enabled.ProviderName() != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got '%s'", enabled.ProviderName())
	}
}

func TestCountAccessible(t *testing.T) {
	validation := []model.ValidationResult{
		{IsAccessible: true},
		{IsAccessible: false},
		{IsAccessible: true},
		{IsAccessible: true},
	}

	count := countAccessible(validation)

	if count != 3 {
		t.Errorf("Expected 3 accessible, got %d", count)
	}
}

func TestCountDead(t *testing.T) {
	validation := []model.ValidationResult{
		{IsAccessible: true},
		{IsAccessible: false},
		{IsAccessible: true},
		{IsAccessible: false},
	}

	count := countDead(validation)

	if count != 2 {
		t.Errorf("Expected 2 dead, got %d", count)
	}
}

func TestJoinURLs_Empty(t *testing.T) {
	result := joinURLs([]string{})

	if !strings.Contains(result, "No evidence URLs available") {
		t.Error("Expected message about no URLs")
	}
}

func TestJoinURLs_Few(t *testing.T) {
	urls := []string{
		"https://example.com/1",
		"https://example.com/2",
	}

	result := joinURLs(urls)

	for _, url := range urls {
		if !strings.Contains(result, url) {
			t.Errorf("Expected result to contain %s", url)
		}
	}
}

// Mock error type for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
