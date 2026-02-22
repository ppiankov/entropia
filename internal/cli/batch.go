package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ppiankov/entropia/internal/model"
	"github.com/ppiankov/entropia/internal/pipeline"
	"github.com/ppiankov/entropia/internal/worker"
	"github.com/spf13/cobra"
)

var (
	concurrency  int
	outputDir    string
	batchTimeout time.Duration
	// noFooter is defined in scan.go and shared here
)

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch <file>",
	Short: "Scan multiple URLs from a file in parallel",
	Long: `Batch processes multiple URLs concurrently:
- Read URLs from input file (one per line)
- Process URLs in parallel with configurable worker count
- Each scan uses concurrent evidence validation
- Generate individual reports for each URL

Example:
  entropia batch urls.txt
  entropia batch urls.txt --concurrency 10 --output-dir ./reports
  entropia batch urls.txt --concurrency 5 --timeout 5m`,
	Args: cobra.ExactArgs(1),
	RunE: runBatch,
}

func init() {
	rootCmd.AddCommand(batchCmd)

	// Concurrency flags
	batchCmd.Flags().IntVar(&concurrency, "concurrency", runtime.NumCPU(), "number of concurrent workers")
	batchCmd.Flags().StringVar(&outputDir, "output-dir", "./entropia-reports", "output directory for reports")
	batchCmd.Flags().DurationVar(&batchTimeout, "timeout", 10*time.Minute, "total timeout for batch processing")

	// Inherit flags from scan command
	batchCmd.Flags().DurationVar(&timeout, "scan-timeout", 30*time.Second, "timeout for individual scans")
	batchCmd.Flags().StringVar(&userAgent, "ua", "Entropia/0.1 (+https://github.com/ppiankov/entropia)", "HTTP User-Agent")
	batchCmd.Flags().BoolVar(&noCache, "no-cache", false, "disable cache (force fresh fetch)")
	batchCmd.Flags().BoolVar(&noFooter, "no-footer", false, "disable footer in Markdown reports")
	batchCmd.Flags().StringVar(&httpProxy, "http-proxy", "", "HTTP proxy URL (overrides HTTP_PROXY env var)")
	batchCmd.Flags().StringVar(&httpsProxy, "https-proxy", "", "HTTPS proxy URL (overrides HTTPS_PROXY env var)")

	// LLM flags
	batchCmd.Flags().BoolVar(&llmEnabled, "llm", false, "enable LLM summary generation")
	batchCmd.Flags().StringVar(&llmProvider, "llm-provider", "openai", "LLM provider (openai, anthropic, ollama)")
	batchCmd.Flags().StringVar(&llmModel, "llm-model", "gpt-4o-mini", "LLM model name")
}

func runBatch(cmd *cobra.Command, args []string) error {
	file := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), batchTimeout)
	defer cancel()

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  Entropia Batch Processing\n")
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Input file:   %s\n", file)
	fmt.Fprintf(os.Stderr, "  Workers:      %d\n", concurrency)
	fmt.Fprintf(os.Stderr, "  Output dir:   %s\n", outputDir)
	fmt.Fprintf(os.Stderr, "  Timeout:      %v\n", batchTimeout)
	fmt.Fprintf(os.Stderr, "\n")

	// Build configuration
	cfg := model.DefaultConfig()
	cfg.HTTP.Timeout = timeout
	cfg.HTTP.UserAgent = userAgent
	cfg.HTTP.HTTPProxy = httpProxy
	cfg.HTTP.HTTPSProxy = httpsProxy
	cfg.Cache.Enabled = !noCache
	cfg.Concurrency.Workers = concurrency
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

		fmt.Fprintf(os.Stderr, "  LLM:          %s/%s\n", llmProvider, llmModel)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Create pipeline
	p := pipeline.NewPipeline(cfg)

	// Create batch processor
	processor := worker.NewBatchProcessor(p, concurrency, cfg.RateLimiting.RequestsPerSecond, cfg.RateLimiting.BurstSize)

	// Process URLs
	fmt.Fprintf(os.Stderr, "⚙️  Reading URLs from file...\n")
	results, err := processor.ProcessFile(ctx, file)
	if err != nil {
		return fmt.Errorf("process file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Loaded %d URLs\n", len(results))
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "⚙️  Processing URLs with %d workers...\n", concurrency)
	fmt.Fprintf(os.Stderr, "\n")

	// Process results
	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Error != nil {
			failureCount++
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", result.URL, result.Error)
			continue
		}

		successCount++

		// Generate output file names
		slug := sanitizeFilename(result.Report.Subject)
		jsonPath := filepath.Join(outputDir, slug+".json")
		mdPath := filepath.Join(outputDir, slug+".md")

		// Render report
		renderer := pipeline.NewRenderer(cfg.Output.IncludeFooter)
		if err := renderer.RenderJSON(result.Report, jsonPath); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: failed to write JSON: %v\n", result.URL, err)
			continue
		}
		if err := renderer.RenderMarkdown(result.Report, mdPath); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: failed to write Markdown: %v\n", result.URL, err)
			continue
		}

		fmt.Fprintf(os.Stderr, "✓ %s (index: %d/100)\n", result.Report.Subject, result.Report.Score.Index)
	}

	// Summary
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "  Batch Complete\n")
	fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Total:     %d URLs\n", len(results))
	fmt.Fprintf(os.Stderr, "  Success:   %d\n", successCount)
	fmt.Fprintf(os.Stderr, "  Failures:  %d\n", failureCount)
	fmt.Fprintf(os.Stderr, "  Output:    %s\n", outputDir)
	fmt.Fprintf(os.Stderr, "\n")

	return nil
}

// sanitizeFilename sanitizes a string for use as a filename
func sanitizeFilename(s string) string {
	s = filepath.Base(s)
	s = filepath.Clean(s)

	// Replace problematic characters
	replacer := []string{
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "-",
	}

	for i := 0; i < len(replacer); i += 2 {
		s = filepath.ToSlash(s)
		s = filepath.Base(s)
	}

	// Limit length
	if len(s) > 100 {
		s = s[:100]
	}

	return s
}
