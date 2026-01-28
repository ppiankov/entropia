package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ppiankov/entropia/internal/model"
	"github.com/ppiankov/entropia/internal/pipeline"
	"github.com/spf13/cobra"
)

var (
	outJSON     string
	outMD       string
	timeout     time.Duration
	userAgent   string
	maxBytes    int64
	noCache     bool
	noFooter    bool
	insecureTLS bool
	llmEnabled  bool
	llmProvider string
	llmModel    string
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan <url>",
	Short: "Scan a single URL and generate entropy/decay report",
	Long: `Scan analyzes a single web page to:
- Extract factual and attributional claims
- Map claims to cited sources
- Evaluate evidence quality, freshness, and consistency
- Detect conflicts, gaps, and decay
- Generate transparent, explainable reports

Example:
  entropia scan https://en.wikipedia.org/wiki/Laksa
  entropia scan https://example.com --json report.json --md report.md
  entropia scan https://example.com --llm openai --model gpt-4o-mini`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Output flags
	scanCmd.Flags().StringVar(&outJSON, "json", "report.json", "output JSON path")
	scanCmd.Flags().StringVar(&outMD, "md", "", "output Markdown path (optional)")

	// HTTP flags
	scanCmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "overall scan timeout (increase for pages with many evidence links)")
	scanCmd.Flags().StringVar(&userAgent, "ua", "Entropia/0.1 (+https://github.com/ppiankov/entropia)", "HTTP User-Agent")
	scanCmd.Flags().Int64Var(&maxBytes, "max-bytes", 2_000_000, "max response bytes to read")
	scanCmd.Flags().BoolVar(&noCache, "no-cache", false, "disable cache (force fresh fetch)")
	scanCmd.Flags().BoolVar(&noFooter, "no-footer", false, "disable footer in Markdown reports")
	scanCmd.Flags().BoolVar(&insecureTLS, "insecure", false, "skip TLS certificate verification (use for self-signed certs)")

	// LLM flags
	scanCmd.Flags().BoolVar(&llmEnabled, "llm", false, "enable LLM summary generation")
	scanCmd.Flags().StringVar(&llmProvider, "llm-provider", "openai", "LLM provider (openai, anthropic, ollama)")
	scanCmd.Flags().StringVar(&llmModel, "llm-model", "gpt-4o-mini", "LLM model name")
}

func runScan(cmd *cobra.Command, args []string) error {
	url := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning: %s\n", url)
		fmt.Fprintf(os.Stderr, "Timeout: %v\n", timeout)
		fmt.Fprintf(os.Stderr, "Cache: %v\n", !noCache)
		fmt.Fprintln(os.Stderr)
	}

	// Build configuration from flags
	cfg := model.DefaultConfig()
	cfg.HTTP.Timeout = timeout
	cfg.HTTP.UserAgent = userAgent
	cfg.HTTP.MaxBodyBytes = maxBytes
	cfg.HTTP.InsecureTLS = insecureTLS
	cfg.Cache.Enabled = !noCache
	cfg.Output.Verbose = verbose
	cfg.Output.IncludeFooter = !noFooter

	// Configure LLM if enabled
	if llmEnabled {
		cfg.LLM.Provider = llmProvider
		cfg.LLM.Model = llmModel
		cfg.LLM.StrictEvidence = true // Always enforce

		// Get API key from environment
		switch llmProvider {
		case "openai":
			cfg.LLM.APIKey = os.Getenv("OPENAI_API_KEY")
			if cfg.LLM.APIKey == "" {
				return fmt.Errorf("OPENAI_API_KEY environment variable not set")
			}
		case "anthropic", "claude":
			cfg.LLM.APIKey = os.Getenv("ANTHROPIC_API_KEY")
			if cfg.LLM.APIKey == "" {
				return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
			}
		case "ollama":
			// Ollama doesn't need an API key
			baseURL := os.Getenv("OLLAMA_BASE_URL")
			if baseURL != "" {
				cfg.LLM.BaseURL = baseURL
			}
		}
	}

	// Create pipeline
	p := pipeline.NewPipeline(cfg)

	// Scan URL
	if verbose {
		fmt.Fprintf(os.Stderr, "⚙️  Fetching HTML...\n")
	}

	result, err := p.ScanURL(ctx, url)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✓ Extracted %d claims\n", len(result.Report.Claims))
		fmt.Fprintf(os.Stderr, "✓ Extracted %d evidence links\n", len(result.Report.Evidence))
		fmt.Fprintf(os.Stderr, "✓ Calculated support index: %d/100\n", result.Report.Score.Index)
		if result.Report.LLM != nil && result.Report.LLM.Enabled {
			fmt.Fprintf(os.Stderr, "✓ Generated LLM summary using %s/%s\n", result.Report.LLM.Provider, result.Report.LLM.Model)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Render outputs
	if err := p.RenderReport(result.Report, outJSON, outMD, verbose); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	return nil
}
