package validate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ppiankov/entropia/internal/model"
)

func init() {
	// Disable retry sleep in all tests for fast execution
	validateSleepFunc = func(d time.Duration) {}
}

func TestValidator_ValidateSingle_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2023 15:04:05 GMT")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{
		URL:  server.URL,
		Kind: model.EvidenceKindExternalLink,
	}

	result := validator.validateSingle(context.Background(), evidence)

	if !result.IsAccessible {
		t.Error("Expected link to be accessible")
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if result.IsDead {
		t.Error("Expected link not to be dead")
	}

	if result.LastModified == nil {
		t.Error("Expected Last-Modified to be parsed")
	}
}

func TestValidator_ValidateSingle_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{URL: server.URL}

	result := validator.validateSingle(context.Background(), evidence)

	if result.IsAccessible {
		t.Error("Expected 404 link not to be accessible")
	}

	if !result.IsDead {
		t.Error("Expected 404 link to be marked as dead")
	}

	if result.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", result.StatusCode)
	}
}

func TestValidator_ValidateSingle_Redirect(t *testing.T) {
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{URL: redirectServer.URL}

	result := validator.validateSingle(context.Background(), evidence)

	if !result.IsAccessible {
		t.Error("Expected redirected link to be accessible")
	}

	if result.RedirectURL == "" {
		t.Error("Expected redirect URL to be captured")
	}

	if result.RedirectURL != finalServer.URL {
		t.Errorf("Expected redirect to %s, got %s", finalServer.URL, result.RedirectURL)
	}
}

func TestValidator_ValidateSingle_Staleness(t *testing.T) {
	tests := []struct {
		lastModified string
		expectStale  bool
		expectVery   bool
		desc         string
	}{
		{
			lastModified: time.Now().Add(-400 * 24 * time.Hour).Format(time.RFC1123),
			expectStale:  true,
			expectVery:   false,
			desc:         "13-month-old source should be stale",
		},
		{
			lastModified: time.Now().Add(-4 * 365 * 24 * time.Hour).Format(time.RFC1123),
			expectStale:  true,
			expectVery:   true,
			desc:         "4-year-old source should be very stale",
		},
		{
			lastModified: time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC1123),
			expectStale:  false,
			expectVery:   false,
			desc:         "30-day-old source should not be stale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Last-Modified", tt.lastModified)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			validator := NewValidator(5*time.Second, 20, nil, "", "", "")
			evidence := model.Evidence{URL: server.URL}

			result := validator.validateSingle(context.Background(), evidence)

			if result.IsStale != tt.expectStale {
				t.Errorf("Expected IsStale=%v, got %v", tt.expectStale, result.IsStale)
			}

			if result.IsVeryStale != tt.expectVery {
				t.Errorf("Expected IsVeryStale=%v, got %v", tt.expectVery, result.IsVeryStale)
			}

			if result.Age == nil {
				t.Error("Expected age to be calculated")
			}
		})
	}
}

func TestValidator_Validate_Concurrency(t *testing.T) {
	// Create multiple test servers
	serverCount := 10
	servers := make([]*httptest.Server, serverCount)
	for i := 0; i < serverCount; i++ {
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond) // Simulate network delay
			w.WriteHeader(http.StatusOK)
		}))
		defer servers[i].Close()
	}

	// Create evidence list
	evidence := make([]model.Evidence, serverCount)
	for i := 0; i < serverCount; i++ {
		evidence[i] = model.Evidence{
			URL:  servers[i].URL,
			Kind: model.EvidenceKindExternalLink,
		}
	}

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")

	start := time.Now()
	results, err := validator.Validate(context.Background(), evidence)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != serverCount {
		t.Errorf("Expected %d results, got %d", serverCount, len(results))
	}

	// With concurrency, 10 requests @ 100ms each should complete in < 500ms
	// Without concurrency, it would take 1000ms
	if duration > 500*time.Millisecond {
		t.Errorf("Validation took too long (%v), concurrent execution may not be working", duration)
	}

	// Check all results are accessible
	for i, result := range results {
		if !result.IsAccessible {
			t.Errorf("Result %d: expected accessible", i)
		}
	}
}

func TestValidator_Validate_EmptyEvidence(t *testing.T) {
	validator := NewValidator(5*time.Second, 20, nil, "", "", "")

	results, err := validator.Validate(context.Background(), []model.Evidence{})

	if err != nil {
		t.Errorf("Expected no error for empty evidence, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty evidence, got %d", len(results))
	}
}

func TestValidator_Validate_ContextCancellation(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	evidence := []model.Evidence{
		{URL: server.URL},
	}

	validator := NewValidator(10*time.Second, 20, nil, "", "", "")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	results, err := validator.Validate(ctx, evidence)

	if err != nil {
		t.Errorf("Expected no error (context cancellation handled gracefully), got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Result should indicate failure due to context cancellation
	if results[0].IsAccessible {
		t.Error("Expected link not to be accessible after context cancellation")
	}
}

func TestValidator_Validate_MixedResults(t *testing.T) {
	// Server 1: OK
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	// Server 2: 404
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server2.Close()

	// Server 3: 500
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server3.Close()

	evidence := []model.Evidence{
		{URL: server1.URL},
		{URL: server2.URL},
		{URL: server3.URL},
	}

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	results, err := validator.Validate(context.Background(), evidence)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Check first result (OK)
	if !results[0].IsAccessible {
		t.Error("Expected first link to be accessible")
	}

	// Check second result (404)
	if results[1].IsAccessible {
		t.Error("Expected second link not to be accessible")
	}
	if !results[1].IsDead {
		t.Error("Expected second link to be marked as dead")
	}

	// Check third result (500)
	if results[2].IsAccessible {
		t.Error("Expected third link not to be accessible (500 error)")
	}
}

func TestValidator_AuthorityClassification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &model.AuthorityConfig{
		PrimaryDomains: []string{"127.0.0.1"},
	}

	validator := NewValidator(5*time.Second, 20, config, "", "", "")
	evidence := model.Evidence{URL: server.URL}

	result := validator.validateSingle(context.Background(), evidence)

	// Server URL will be 127.0.0.1 (localhost), which we configured as primary
	if result.Authority != model.TierPrimary {
		t.Errorf("Expected authority tier to be primary, got %v", result.Authority)
	}
}

func TestNewValidator_DefaultWorkers(t *testing.T) {
	validator := NewValidator(5*time.Second, 0, nil, "", "", "")

	if validator.maxWorkers != 20 {
		t.Errorf("Expected default max workers to be 20, got %d", validator.maxWorkers)
	}
}

func TestNewValidator_CustomWorkers(t *testing.T) {
	validator := NewValidator(5*time.Second, 50, nil, "", "", "")

	if validator.maxWorkers != 50 {
		t.Errorf("Expected max workers to be 50, got %d", validator.maxWorkers)
	}
}

func TestValidateSingleWithRetry_TransientThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{URL: server.URL}

	result := validator.validateSingleWithRetry(context.Background(), evidence)

	if !result.IsAccessible {
		t.Error("Expected accessible after retry")
	}
	if attempts.Load() != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts.Load())
	}
}

func TestValidateSingleWithRetry_PermanentFailure(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{URL: server.URL}

	result := validator.validateSingleWithRetry(context.Background(), evidence)

	if result.IsAccessible {
		t.Error("Expected not accessible for 404")
	}
	if !result.IsDead {
		t.Error("Expected dead for 404")
	}
	// 404 is not retryable — should only attempt once
	if attempts.Load() != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts.Load())
	}
}

func TestValidateSingleWithRetry_AllRetriesExhausted(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{URL: server.URL}

	result := validator.validateSingleWithRetry(context.Background(), evidence)

	if result.IsAccessible {
		t.Error("Expected not accessible after all retries exhausted")
	}
	if attempts.Load() != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts.Load())
	}
}

func TestValidateSingleWithRetry_429Retried(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewValidator(5*time.Second, 20, nil, "", "", "")
	evidence := model.Evidence{URL: server.URL}

	result := validator.validateSingleWithRetry(context.Background(), evidence)

	if !result.IsAccessible {
		t.Error("Expected accessible after 429 retry")
	}
	if attempts.Load() != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts.Load())
	}
}

func TestIsRetryableValidationResult(t *testing.T) {
	tests := []struct {
		desc      string
		result    model.ValidationResult
		retryable bool
	}{
		{"200 OK", model.ValidationResult{StatusCode: 200, IsAccessible: true}, false},
		{"404 Not Found", model.ValidationResult{StatusCode: 404, IsDead: true}, false},
		{"500 Server Error", model.ValidationResult{StatusCode: 500}, true},
		{"502 Bad Gateway", model.ValidationResult{StatusCode: 502}, true},
		{"503 Service Unavailable", model.ValidationResult{StatusCode: 503}, true},
		{"429 Too Many Requests", model.ValidationResult{StatusCode: 429}, true},
		{"timeout error", model.ValidationResult{Error: "request failed: timeout"}, true},
		{"connection refused", model.ValidationResult{Error: "request failed: connection refused"}, true},
		{"create request error", model.ValidationResult{Error: "create request: invalid URL"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := isRetryableValidationResult(tt.result)
			if got != tt.retryable {
				t.Errorf("isRetryableValidationResult(%s) = %v, want %v", tt.desc, got, tt.retryable)
			}
		})
	}
}

func TestValidateBatch_ConvenienceFunction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	evidence := []model.Evidence{
		{URL: server.URL},
	}

	results, err := ValidateBatch(context.Background(), evidence, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].IsAccessible {
		t.Error("Expected link to be accessible")
	}
}
