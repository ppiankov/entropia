package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ppiankov/entropia/internal/extract"
	"github.com/ppiankov/entropia/internal/llm"
	"github.com/ppiankov/entropia/internal/model"
	"github.com/ppiankov/entropia/internal/score"
	"github.com/ppiankov/entropia/internal/validate"
)

// Pipeline orchestrates the complete scan process
type Pipeline struct {
	fetcher        *Fetcher
	claimExtractor *extract.ClaimExtractor
	evidExtractor  *extract.EvidenceExtractor
	validator      *validate.Validator
	scorer         *score.Scorer
	renderer       *Renderer
	summarizer     *llm.Summarizer // Optional LLM summarizer (nil if disabled)
	config         *model.Config
}

// NewPipeline creates a new pipeline with the given configuration
func NewPipeline(cfg *model.Config) *Pipeline {
	// Create LLM summarizer if configured
	var summarizer *llm.Summarizer
	if cfg.LLM.Provider != "" {
		llmConfig := llm.ConfigFromModel(cfg.LLM)
		s, err := llm.NewSummarizer(llmConfig)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize LLM provider: %v\n", err)
		} else {
			summarizer = s
		}
	}

	return &Pipeline{
		fetcher:        NewFetcher(cfg.HTTP.Timeout, cfg.HTTP.UserAgent, cfg.HTTP.MaxBodyBytes),
		claimExtractor: extract.NewClaimExtractor(),
		evidExtractor:  extract.NewEvidenceExtractor(),
		validator:      validate.NewValidator(10*time.Second, cfg.Concurrency.ValidationWorkers, &cfg.Authority),
		scorer:         score.NewScorer(),
		renderer:       NewRenderer(cfg.Output.IncludeFooter),
		summarizer:     summarizer,
		config:         cfg,
	}
}

// ScanResult contains the complete scan result
type ScanResult struct {
	Report *model.Report
	Error  error
}

// ScanURL scans a single URL and generates a complete report
func (p *Pipeline) ScanURL(ctx context.Context, url string) (*ScanResult, error) {
	// 1. Fetch HTML
	fetchResult, err := p.fetcher.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// 2. Extract claims
	claims, err := p.claimExtractor.Extract(fetchResult.HTML)
	if err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}

	// 3. Extract evidence
	evidence, err := p.evidExtractor.Extract(fetchResult.HTML, fetchResult.FinalURL)
	if err != nil {
		return nil, fmt.Errorf("extract evidence: %w", err)
	}

	// 4. Validate evidence concurrently
	validation, err := p.validator.Validate(ctx, evidence)
	if err != nil {
		return nil, fmt.Errorf("validate evidence: %w", err)
	}

	// 5. Calculate score
	scoreResult := p.scorer.Calculate(claims, evidence, validation)

	// 6. Build report (without LLM summary yet)
	report := &model.Report{
		Subject:    fetchResult.Subject,
		SourceURL:  fetchResult.FinalURL,
		FetchedAt:  time.Now().UTC(),
		FetchMeta:  fetchResult.Meta,
		Claims:     claims,
		Evidence:   evidence,
		Validation: validation,
		Score:      scoreResult,
		Principles: model.DefaultPrinciples(),
	}

	// 7. Generate LLM summary if enabled (AFTER scoring, never affects score)
	if p.summarizer != nil && p.summarizer.IsEnabled() {
		llmSummary, err := p.summarizer.GenerateSummary(ctx, *report)
		if err != nil {
			// Don't fail the entire scan, just warn
			fmt.Printf("Warning: LLM summary generation failed: %v\n", err)
		} else if llmSummary != nil {
			report.LLM = llmSummary
		}
	}

	return &ScanResult{
		Report: report,
		Error:  nil,
	}, nil
}


// RenderReport renders the report to the specified outputs
func (p *Pipeline) RenderReport(report *model.Report, jsonPath string, mdPath string, verbose bool) error {
	// Render JSON
	if jsonPath != "" {
		if err := p.renderer.RenderJSON(report, jsonPath); err != nil {
			return fmt.Errorf("render JSON: %w", err)
		}
		if verbose {
			fmt.Printf("✓ Wrote JSON: %s\n", jsonPath)
		}
	}

	// Render Markdown
	if mdPath != "" {
		if err := p.renderer.RenderMarkdown(report, mdPath); err != nil {
			return fmt.Errorf("render markdown: %w", err)
		}
		if verbose {
			fmt.Printf("✓ Wrote Markdown: %s\n", mdPath)
		}
	}

	// Render LLM summary to separate file if present
	if report.LLM != nil && report.LLM.Enabled && mdPath != "" {
		llmMdPath := strings.TrimSuffix(mdPath, ".md") + ".llm.md"
		llmMarkdown := llm.RenderSeparateMarkdown(report.LLM)
		if err := p.renderer.RenderLLMMarkdown(llmMarkdown, llmMdPath); err != nil {
			fmt.Printf("Warning: Failed to write LLM summary: %v\n", err)
		} else if verbose {
			fmt.Printf("✓ Wrote LLM Summary: %s\n", llmMdPath)
		}
	}

	// Print summary to stdout
	p.renderer.RenderSummary(report)

	return nil
}
