package worker

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ppiankov/entropia/internal/model"
	"github.com/ppiankov/entropia/internal/pipeline"
)

// ScanJob represents a URL scan job
type ScanJob struct {
	URL      string
	Pipeline *pipeline.Pipeline
}

// Execute executes the scan job
func (j *ScanJob) Execute(ctx context.Context) Result {
	result, err := j.Pipeline.ScanURL(ctx, j.URL)
	if err != nil {
		return &ScanResult{
			URL:    j.URL,
			Report: nil,
			Error:  err,
		}
	}
	return &ScanResult{
		URL:    j.URL,
		Report: result.Report,
		Error:  nil,
	}
}

// ScanResult represents the result of a scan job
type ScanResult struct {
	URL    string
	Report *model.Report
	Error  error
}

// GetError returns the error from the scan result
func (r *ScanResult) GetError() error {
	return r.Error
}

// BatchProcessor processes multiple URLs concurrently
type BatchProcessor struct {
	pipeline    *pipeline.Pipeline
	concurrency int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(pipeline *pipeline.Pipeline, concurrency int) *BatchProcessor {
	return &BatchProcessor{
		pipeline:    pipeline,
		concurrency: concurrency,
	}
}

// ProcessURLs processes multiple URLs concurrently
func (b *BatchProcessor) ProcessURLs(ctx context.Context, urls []string) []*ScanResult {
	if len(urls) == 0 {
		return []*ScanResult{}
	}

	// Create worker pool
	pool := NewPool(b.concurrency)
	pool.Start()

	// Submit jobs
	for _, url := range urls {
		job := &ScanJob{
			URL:      url,
			Pipeline: b.pipeline,
		}
		pool.Submit(job)
	}

	// Wait for all jobs to complete
	results := pool.Wait()

	// Convert to ScanResults
	scanResults := make([]*ScanResult, len(results))
	for i, result := range results {
		scanResults[i] = result.(*ScanResult)
	}

	return scanResults
}

// ProcessFile reads URLs from a file and processes them concurrently
func (b *BatchProcessor) ProcessFile(ctx context.Context, filePath string) ([]*ScanResult, error) {
	urls, err := ReadURLsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read URLs: %w", err)
	}

	return b.ProcessURLs(ctx, urls), nil
}

// ReadURLsFromFile reads URLs from a file (one per line)
func ReadURLsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var urls []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Deduplicate URLs
		if !seen[line] {
			seen[line] = true
			urls = append(urls, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return urls, nil
}
