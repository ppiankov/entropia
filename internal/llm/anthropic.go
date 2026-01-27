package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude models
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	config     Config
}

// Anthropic API structures
type anthropicRequest struct {
	Model       string              `json:"model"`
	MaxTokens   int                 `json:"max_tokens"`
	Messages    []anthropicMessage  `json:"messages"`
	System      string              `json:"system,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config Config) (*AnthropicProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &AnthropicProvider{
		apiKey:  config.APIKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config: config,
	}, nil
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// IsAvailable checks if the provider is properly configured
func (p *AnthropicProvider) IsAvailable(ctx context.Context) bool {
	// Simple check: make a minimal API call
	// We'll try to get a very short completion
	req := anthropicRequest{
		Model:     "claude-3-5-haiku-20241022",
		MaxTokens: 10,
		Messages: []anthropicMessage{
			{Role: "user", Content: "Hi"},
		},
	}

	_, err := p.makeRequest(ctx, req)
	if err != nil {
		// Log the actual error for debugging (this helps users diagnose API key issues)
		fmt.Fprintf(os.Stderr, "Anthropic API check failed: %v\n", err)
		return false
	}
	return true
}

// Summarize generates a summary using Anthropic's Messages API
func (p *AnthropicProvider) Summarize(ctx context.Context, req SummarizeRequest) (*SummarizeResponse, error) {
	// Build prompt if not provided
	prompt := req.Prompt
	if prompt == "" {
		prompt = BuildPrompt(req.Report, req.EvidenceURLs)
	}

	// Determine model
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Default to Sonnet
	}

	// Determine max tokens
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens
	}
	if maxTokens == 0 {
		maxTokens = 1000
	}

	// Construct API request
	apiReq := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    "You are a helpful assistant that summarizes Entropia reports with strict adherence to evidence constraints.",
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3, // Lower temperature for more focused output
	}

	// Make API call
	resp, err := p.makeRequest(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic API error: %w", err)
	}

	// Extract text from response
	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("no content in Anthropic response")
	}

	summary := strings.TrimSpace(resp.Content[0].Text)

	// Extract URLs from the summary
	citedURLs := extractURLs(summary)

	// CRITICAL: Verify strict evidence mode
	if p.config.StrictEvidence {
		for _, citedURL := range citedURLs {
			if !contains(req.EvidenceURLs, citedURL) {
				return nil, fmt.Errorf("CITATION LEAK: LLM cited disallowed URL: %s", citedURL)
			}
		}
	}

	totalTokens := resp.Usage.InputTokens + resp.Usage.OutputTokens

	return &SummarizeResponse{
		Summary:    summary,
		CitedURLs:  citedURLs,
		Model:      resp.Model,
		TokensUsed: totalTokens,
	}, nil
}

// makeRequest makes an HTTP request to the Anthropic API
func (p *AnthropicProvider) makeRequest(ctx context.Context, apiReq anthropicRequest) (*anthropicResponse, error) {
	// Serialize request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/messages", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		var apiErr anthropicError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API error (%d): %s - %s", httpResp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var resp anthropicResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}
