package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ppiankov/entropia/internal/model"
	"github.com/sashabaranov/go-openai"
)

func TestOpenAIProvider_Summarize_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		// Return success response
		resp := openai.ChatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4o-mini",
			Choices: []openai.ChatCompletionChoice{
				{
					Index: 0,
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "This is a summary. Source: https://example.com/1",
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.Usage{
				TotalTokens: 100,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider
	config := Config{
		APIKey:         "test-key",
		BaseURL:        server.URL,
		Model:          "gpt-4o-mini",
		Timeout:        5,
		StrictEvidence: true,
	}
	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test Summarize
	req := SummarizeRequest{
		Report: model.Report{Subject: "Test"},
		EvidenceURLs: []string{
			"https://example.com/1",
		},
	}

	resp, err := provider.Summarize(context.Background(), req)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if resp.Summary != "This is a summary. Source: https://example.com/1" {
		t.Errorf("Unexpected summary: %s", resp.Summary)
	}
	if len(resp.CitedURLs) != 1 || resp.CitedURLs[0] != "https://example.com/1" {
		t.Errorf("Unexpected cited URLs: %v", resp.CitedURLs)
	}
}

func TestOpenAIProvider_Summarize_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"message": "Internal Server Error", "type": "server_error"}}`))
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := SummarizeRequest{
		Report: model.Report{Subject: "Test"},
	}

	_, err = provider.Summarize(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestOpenAIProvider_Summarize_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`))
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := SummarizeRequest{
		Report: model.Report{Subject: "Test"},
	}

	_, err = provider.Summarize(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestOpenAIProvider_Summarize_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{malformed json`))
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := SummarizeRequest{
		Report: model.Report{Subject: "Test"},
	}

	_, err = provider.Summarize(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for malformed JSON, got nil")
	}
}

func TestOpenAIProvider_Summarize_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate delay
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set timeout shorter than delay
	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 1, // 1 second? Wait, config.Timeout is int seconds.
	}
	// The provider uses time.Duration(p.config.Timeout) * time.Second.
	// But 1 second is too long for a unit test to wait if we want it fast,
	// and sleep(100ms) is too short.
	// However, the test uses context.WithTimeout.

	// Let's use a mock transport or just rely on the fact that we can set timeout in config.
	// But the provider implementation is:
	// timeout := time.Duration(p.config.Timeout) * time.Second
	// if timeout == 0 { timeout = 30 * time.Second }
	// So minimum timeout is 1 second if config.Timeout = 1.

	// To properly test timeout without waiting 1 second, we'd need to mock the client creation
	// or use a very short timeout. But `config.Timeout` is int (seconds).
	// So we have to set it to 1 second and sleep for 1.1s in the server?
	// That slows down tests.

	// Alternative: pass a context with a short deadline to Summarize.
	// Summarize signature: Summarize(ctx context.Context, req SummarizeRequest)
	// Inside Summarize:
	// ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	//
	// If the passed ctx is already cancelled or has a shorter deadline, it should be respected?
	// The code:
	// ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	//
	// `WithTimeout` returns a context that is canceled when the parent is canceled OR when the timeout expires.
	// So if we pass a context with a very short timeout, it should work.

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req := SummarizeRequest{
		Report: model.Report{Subject: "Test"},
	}

	// We need the server to hang longer than 10ms
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	_, err = provider.Summarize(ctx, req)
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestOpenAIProvider_IsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": [{"id": "gpt-4o-mini"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}
	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if !provider.IsAvailable(context.Background()) {
		t.Error("Expected available to be true")
	}

	// Test failure
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if provider.IsAvailable(context.Background()) {
		t.Error("Expected available to be false on error")
	}
}
