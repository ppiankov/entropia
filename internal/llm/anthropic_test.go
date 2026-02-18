package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ppiankov/entropia/internal/model"
)

func TestAnthropicProvider_Summarize_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/v1/messages" {
			t.Errorf("Expected path /v1/messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header test-key, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header 2023-06-01, got %s", r.Header.Get("anthropic-version"))
		}

		// Return success response
		resp := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{
					Type: "text",
					Text: "This is a summary. Source: https://example.com/1",
				},
			},
			Model: "claude-3-5-sonnet-20241022",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  50,
				OutputTokens: 50,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider
	config := Config{
		APIKey:         "test-key",
		BaseURL:        server.URL,
		Model:          "claude-3-5-sonnet-20241022",
		Timeout:        5,
		StrictEvidence: true,
	}
	provider, err := NewAnthropicProvider(config)
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
	if resp.TokensUsed != 100 {
		t.Errorf("Unexpected token usage: %d", resp.TokensUsed)
	}
}

func TestAnthropicProvider_Summarize_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type": "error", "error": {"type": "api_error", "message": "Internal Server Error"}}`))
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	provider, err := NewAnthropicProvider(config)
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
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("Expected error message to contain 'Internal Server Error', got %v", err)
	}
}

func TestAnthropicProvider_Summarize_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"type": "error", "error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	provider, err := NewAnthropicProvider(config)
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

func TestAnthropicProvider_Summarize_MalformedJSON(t *testing.T) {
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
	provider, err := NewAnthropicProvider(config)
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

func TestAnthropicProvider_IsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock a successful response for the minimal check
		resp := anthropicResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{Type: "text", Text: "Hi"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}
	provider, err := NewAnthropicProvider(config)
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
