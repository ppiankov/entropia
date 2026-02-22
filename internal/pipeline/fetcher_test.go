package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchWithRetry_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, "<html><body>OK</body></html>")
	}))
	defer server.Close()

	fetcher := NewFetcher(5*time.Second, "test-agent", 1<<20, false, "", "", "")
	result, err := fetcher.FetchWithRetry(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result.HTML != "<html><body>OK</body></html>" {
		t.Errorf("Unexpected HTML: %s", result.HTML)
	}
}

func TestFetchWithRetry_TransientThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, "<html>OK</html>")
	}))
	defer server.Close()

	// Override sleep for fast tests
	origSleep := fetchSleepFunc
	fetchSleepFunc = func(d time.Duration) {}
	defer func() { fetchSleepFunc = origSleep }()

	fetcher := NewFetcher(5*time.Second, "test-agent", 1<<20, false, "", "", "")
	result, err := fetcher.FetchWithRetry(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Expected success after retries, got %v", err)
	}
	if result.HTML != "<html>OK</html>" {
		t.Errorf("Unexpected HTML: %s", result.HTML)
	}
	if attempts.Load() != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts.Load())
	}
}

func TestFetchWithRetry_PermanentFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	origSleep := fetchSleepFunc
	fetchSleepFunc = func(d time.Duration) {}
	defer func() { fetchSleepFunc = origSleep }()

	fetcher := NewFetcher(5*time.Second, "test-agent", 1<<20, false, "", "", "")
	_, err := fetcher.FetchWithRetry(context.Background(), server.URL)
	if err == nil {
		t.Fatal("Expected error for 404, got nil")
	}
	// 404 is not retryable, so should fail immediately
	if got := err.Error(); got != "unexpected status: 404 404 Not Found" {
		t.Errorf("Unexpected error: %s", got)
	}
}

func TestFetchWithRetry_AllRetriesExhausted(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	origSleep := fetchSleepFunc
	fetchSleepFunc = func(d time.Duration) {}
	defer func() { fetchSleepFunc = origSleep }()

	fetcher := NewFetcher(5*time.Second, "test-agent", 1<<20, false, "", "", "")
	_, err := fetcher.FetchWithRetry(context.Background(), server.URL)
	if err == nil {
		t.Fatal("Expected error after all retries exhausted")
	}
	if attempts.Load() != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts.Load())
	}
}

func TestFetchWithRetry_429Retried(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, "<html>OK</html>")
	}))
	defer server.Close()

	origSleep := fetchSleepFunc
	fetchSleepFunc = func(d time.Duration) {}
	defer func() { fetchSleepFunc = origSleep }()

	fetcher := NewFetcher(5*time.Second, "test-agent", 1<<20, false, "", "", "")
	result, err := fetcher.FetchWithRetry(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Expected success after 429 retry, got %v", err)
	}
	if result.HTML != "<html>OK</html>" {
		t.Errorf("Unexpected HTML: %s", result.HTML)
	}
	if attempts.Load() != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts.Load())
	}
}

func TestIsRetryableFetchError(t *testing.T) {
	tests := []struct {
		err       string
		retryable bool
	}{
		{"unexpected status: 503 Service Unavailable", true},
		{"unexpected status: 500 Internal Server Error", true},
		{"unexpected status: 502 Bad Gateway", true},
		{"unexpected status: 429 Too Many Requests", true},
		{"unexpected status: 404 Not Found", false},
		{"unexpected status: 403 Forbidden", false},
		{"unexpected status: 401 Unauthorized", false},
		{"fetch: connection refused", true},
		{"fetch: connection reset by peer", true},
		{"create request: invalid URL", false},
		{"read body: unexpected EOF", false},
	}

	for _, tt := range tests {
		t.Run(tt.err, func(t *testing.T) {
			err := fmt.Errorf("%s", tt.err)
			got := isRetryableFetchError(err)
			if got != tt.retryable {
				t.Errorf("isRetryableFetchError(%q) = %v, want %v", tt.err, got, tt.retryable)
			}
		})
	}
}

func TestIsRetryableFetchError_Nil(t *testing.T) {
	if isRetryableFetchError(nil) {
		t.Error("Expected nil error to not be retryable")
	}
}
