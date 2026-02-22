package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ppiankov/entropia/internal/cache"
	"github.com/ppiankov/entropia/internal/extract"
	"github.com/ppiankov/entropia/internal/extract/adapters"
	"github.com/ppiankov/entropia/internal/llm"
	"github.com/ppiankov/entropia/internal/model"
	"github.com/ppiankov/entropia/internal/score"
	"github.com/ppiankov/entropia/internal/validate"
	"golang.org/x/net/html"
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
	cache          *cache.LayeredCache
	config         *model.Config
}

// NewPipeline creates a new pipeline with the given configuration
func NewPipeline(cfg *model.Config) *Pipeline {
	// Create LLM summarizer if configured
	var summarizer *llm.Summarizer
	if cfg.LLM.Provider != "" {
		// Copy proxy settings from HTTP config to LLM config
		llmModelCfg := cfg.LLM
		llmModelCfg.HTTPProxy = cfg.HTTP.HTTPProxy
		llmModelCfg.HTTPSProxy = cfg.HTTP.HTTPSProxy
		llmModelCfg.NoProxy = cfg.HTTP.NoProxy

		llmConfig := llm.ConfigFromModel(llmModelCfg)
		s, err := llm.NewSummarizer(llmConfig)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize LLM provider: %v\n", err)
		} else {
			summarizer = s
		}
	}

	// Initialize cache if enabled
	var lc *cache.LayeredCache
	if cfg.Cache.Enabled {
		cacheDir := cfg.Cache.Dir
		if strings.HasPrefix(cacheDir, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				cacheDir = home + cacheDir[1:]
			}
		}
		lc = cache.NewLayeredCache(cfg.Cache.TTL, cacheDir, cfg.Cache.TTL)
	}

	return &Pipeline{
		fetcher:        NewFetcher(cfg.HTTP.Timeout, cfg.HTTP.UserAgent, cfg.HTTP.MaxBodyBytes, cfg.HTTP.InsecureTLS, cfg.HTTP.HTTPProxy, cfg.HTTP.HTTPSProxy, cfg.HTTP.NoProxy),
		claimExtractor: extract.NewClaimExtractor(),
		evidExtractor:  extract.NewEvidenceExtractor(),
		validator:      validate.NewValidator(10*time.Second, cfg.Concurrency.ValidationWorkers, &cfg.Authority, cfg.HTTP.HTTPProxy, cfg.HTTP.HTTPSProxy, cfg.HTTP.NoProxy),
		scorer:         score.NewScorer(),
		renderer:       NewRenderer(cfg.Output.IncludeFooter),
		summarizer:     summarizer,
		cache:          lc,
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
	// Check cache first
	if p.cache != nil {
		key := cache.CacheKey(url)
		if data, found := p.cache.Get(key); found {
			var report model.Report
			if err := json.Unmarshal(data, &report); err == nil {
				return &ScanResult{Report: &report}, nil
			}
		}
	}

	// 1. Fetch HTML
	fetchResult, err := p.fetcher.FetchWithRetry(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	// Generate TLS-related signals
	tlsSignals := p.generateTLSSignals(fetchResult.FinalURL, fetchResult.Meta.TLS)

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

	// Append TLS signals to score
	scoreResult.Signals = append(scoreResult.Signals, tlsSignals...)

	// 6. Detect Wikipedia-specific conflicts (edit wars, historical entities)
	if strings.Contains(fetchResult.FinalURL, "wikipedia.org") {
		doc, err := html.Parse(strings.NewReader(fetchResult.HTML))
		if err == nil {
			adapter := adapters.NewWikipediaAdapter()

			// Create a separate context with shorter timeout for conflict detection
			conflictCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			conflictSignals := adapter.DetectWikipediaConflicts(conflictCtx, fetchResult.FinalURL, fetchResult.HTML, doc)
			// Append conflict signals to score
			scoreResult.Signals = append(scoreResult.Signals, conflictSignals...)
		}
	}

	// 7. Build report (without LLM summary yet)
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

	// 8. Store in cache (before LLM summary — cache the deterministic result)
	if p.cache != nil {
		if data, err := json.Marshal(report); err == nil {
			key := cache.CacheKey(url)
			_ = p.cache.Set(key, data, p.config.Cache.TTL)
		}
	}

	// 9. Generate LLM summary if enabled (AFTER scoring, never affects score)
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

// generateTLSSignals creates signals for TLS/certificate issues
func (p *Pipeline) generateTLSSignals(url string, tls *model.TLSInfo) []model.Signal {
	var signals []model.Signal

	if tls == nil {
		return signals
	}

	// 1. No TLS (HTTP only)
	if !tls.Enabled {
		signals = append(signals, model.Signal{
			Type:        model.SignalNoTLS,
			Severity:    model.SeverityWarning,
			Description: "Page served over HTTP without encryption",
			Data: map[string]interface{}{
				"url":         url,
				"explanation": "HTTP connections are unencrypted and vulnerable to tampering. Evidence from unencrypted sources is less trustworthy.",
			},
		})
		return signals // No other TLS checks needed
	}

	// 2. Expired certificate
	if tls.Expired {
		signals = append(signals, model.Signal{
			Type:        model.SignalExpiredCertificate,
			Severity:    model.SeverityCritical,
			Description: "TLS certificate expired or not yet valid",
			Data: map[string]interface{}{
				"subject":     tls.Subject,
				"not_before":  tls.NotBefore,
				"not_after":   tls.NotAfter,
				"explanation": "Expired certificates suggest the site is not actively maintained, raising questions about content freshness.",
			},
		})
	}

	// 3. Self-signed certificate
	if tls.SelfSigned {
		signals = append(signals, model.Signal{
			Type:        model.SignalSelfSignedCertificate,
			Severity:    model.SeverityWarning,
			Description: "TLS certificate is self-signed",
			Data: map[string]interface{}{
				"issuer":      tls.Issuer,
				"subject":     tls.Subject,
				"explanation": "Self-signed certificates cannot be verified by trusted authorities, indicating lower trust.",
			},
		})
	}

	// 4. Domain mismatch
	if tls.DomainMismatch {
		signals = append(signals, model.Signal{
			Type:        model.SignalCertificateMismatch,
			Severity:    model.SeverityCritical,
			Description: "TLS certificate domain doesn't match URL",
			Data: map[string]interface{}{
				"url":         url,
				"dns_names":   tls.DNSNames,
				"explanation": "Certificate issued for a different domain suggests misconfiguration or potential security risk.",
			},
		})
	}

	return signals
}
