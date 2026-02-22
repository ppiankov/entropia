package worker

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ppiankov/entropia/internal/model"
	"github.com/ppiankov/entropia/internal/pipeline"
)

// MockScanner implements Scanner interface
type MockScanner struct {
	ShouldError bool
}

func (m *MockScanner) ScanURL(ctx context.Context, url string) (*pipeline.ScanResult, error) {
	time.Sleep(10 * time.Millisecond) // Simulate work
	if m.ShouldError {
		return nil, errors.New("scan error")
	}
	return &pipeline.ScanResult{
		Report: &model.Report{
			Subject:   "Test Subject",
			SourceURL: url,
		},
	}, nil
}

func TestBatchProcessor_ProcessURLs(t *testing.T) {
	scanner := &MockScanner{}
	processor := NewBatchProcessor(scanner, 2, 0, 0)

	urls := []string{"http://example.com", "http://google.com", "http://bing.com"}
	ctx := context.Background()

	results := processor.ProcessURLs(ctx, urls)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	successCount := 0
	for _, res := range results {
		if res.Error == nil {
			successCount++
			if res.Report == nil {
				t.Error("expected report for successful scan")
			}
		} else {
			t.Errorf("unexpected error for %s: %v", res.URL, res.Error)
		}
	}

	if successCount != 3 {
		t.Errorf("expected 3 successes, got %d", successCount)
	}
}

func TestBatchProcessor_ProcessURLs_Error(t *testing.T) {
	scanner := &MockScanner{ShouldError: true}
	processor := NewBatchProcessor(scanner, 2, 0, 0)

	urls := []string{"http://example.com"}
	ctx := context.Background()

	results := processor.ProcessURLs(ctx, urls)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("expected error, got nil")
	}
	if results[0].Report != nil {
		t.Error("expected nil report on error")
	}
}

func TestBatchProcessor_ProcessURLs_Empty(t *testing.T) {
	scanner := &MockScanner{}
	processor := NewBatchProcessor(scanner, 2, 0, 0)

	results := processor.ProcessURLs(context.Background(), []string{})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestReadURLsFromFile(t *testing.T) {
	content := `http://example.com
# comment
https://google.com
   
http://bing.com   `

	tmpfile, err := os.CreateTemp("", "urls")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	urls, err := ReadURLsFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadURLsFromFile failed: %v", err)
	}

	expected := []string{"http://example.com", "https://google.com", "http://bing.com"}
	if len(urls) != len(expected) {
		t.Fatalf("expected %d URLs, got %d", len(expected), len(urls))
	}

	for i, url := range urls {
		if url != expected[i] {
			t.Errorf("expected URL %s at index %d, got %s", expected[i], i, url)
		}
	}
}

func TestReadURLsFromFile_NonExistent(t *testing.T) {
	_, err := ReadURLsFromFile("non_existent_file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestScanResult_GetError(t *testing.T) {
	r1 := &ScanResult{URL: "http://example.com", Error: nil}
	if r1.GetError() != nil {
		t.Errorf("expected nil error, got %v", r1.GetError())
	}

	expected := errors.New("scan failed")
	r2 := &ScanResult{URL: "http://example.com", Error: expected}
	if r2.GetError() != expected {
		t.Errorf("expected %v, got %v", expected, r2.GetError())
	}
}

func TestBatchProcessor_ProcessFile(t *testing.T) {
	content := "http://example.com\nhttps://google.com\n# comment\n\nhttp://bing.com\n"

	tmpfile, err := os.CreateTemp("", "batch_urls")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	scanner := &MockScanner{}
	processor := NewBatchProcessor(scanner, 2, 0, 0)

	results, err := processor.ProcessFile(context.Background(), tmpfile.Name())
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestBatchProcessor_ProcessFile_NonExistent(t *testing.T) {
	scanner := &MockScanner{}
	processor := NewBatchProcessor(scanner, 2, 0, 0)

	_, err := processor.ProcessFile(context.Background(), "no_such_file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestBatchProcessor_ProcessFile_Empty(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "empty_urls")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	scanner := &MockScanner{}
	processor := NewBatchProcessor(scanner, 2, 0, 0)

	results, err := processor.ProcessFile(context.Background(), tmpfile.Name())
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty file, got %d", len(results))
	}
}

func TestReadURLsFromFile_Deduplication(t *testing.T) {
	content := `http://example.com
http://example.com`

	tmpfile, err := os.CreateTemp("", "urls_dedup")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	urls, err := ReadURLsFromFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadURLsFromFile failed: %v", err)
	}

	if len(urls) != 1 {
		t.Errorf("expected 1 URL after deduplication, got %d", len(urls))
	}
}
