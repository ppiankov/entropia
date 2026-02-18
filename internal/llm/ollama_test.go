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

func TestOllamaProvider_Summarize_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected path /api/generate, got %s", r.URL.Path)
		}

		// Return success response
		resp := ollamaResponse{
			Model:    "llama3.1",
			Response: "This is a summary. Source: https://example.com/1",
			Done:     true,
			PromptEvalCount: 10,
			EvalCount:       20,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider
	config := Config{
		BaseURL:        server.URL,
		Model:          "llama3.1",
		Timeout:        5,
		StrictEvidence: true,
	}
	provider, err := NewOllamaProvider(config)
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
	if resp.TokensUsed != 30 {
		t.Errorf("Unexpected token usage: %d", resp.TokensUsed)
	}
}

func TestOllamaProvider_Summarize_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	config := Config{
		BaseURL: server.URL,
		Model:   "llama3.1",
		Timeout: 5,
	}
	provider, err := NewOllamaProvider(config)
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

func TestOllamaProvider_Summarize_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{malformed json`))
	}))
	defer server.Close()

	config := Config{
		BaseURL: server.URL,
		Model:   "llama3.1",
		Timeout: 5,
	}
	provider, err := NewOllamaProvider(config)
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

func TestOllamaProvider_IsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := Config{
		BaseURL: server.URL,
	}
	provider, err := NewOllamaProvider(config)
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

func TestOllamaProvider_Summarize_NoModel(t *testing.T) {
	config := Config{
		BaseURL: "http://localhost:11434",
		Model:   "", // No model
	}
	provider, err := NewOllamaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req := SummarizeRequest{
		Report: model.Report{Subject: "Test"},
	}

	_, err = provider.Summarize(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error when no model provided, got nil")
	}
	if !strings.Contains(err.Error(), "must be specified") {
		t.Errorf("Expected error about missing model, got %v", err)
	}
}
