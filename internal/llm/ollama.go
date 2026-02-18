package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func newOllamaProxyFunc(httpProxy, httpsProxy, noProxy string) func(*http.Request) (*url.URL, error) {
	if httpProxy == "" && httpsProxy == "" {
		return http.ProxyFromEnvironment
	}

	return func(req *http.Request) (*url.URL, error) {
		if req.URL.Scheme == "https" && httpsProxy != "" {
			return url.Parse(httpsProxy)
		}
		if httpProxy != "" {
			return url.Parse(httpProxy)
		}
		return http.ProxyFromEnvironment(req)
	}
}

// OllamaProvider implements the Provider interface for Ollama local models
type OllamaProvider struct {
	baseURL    string
	httpClient *http.Client
	config     Config
}

// Ollama API structures
type ollamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	System  string        `json:"system,omitempty"`
	Options ollamaOptions `json:"options,omitempty"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // Max tokens
}

type ollamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`

	// Token counts (only present when done=true)
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

type ollamaError struct {
	Error string `json:"error"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config Config) (*OllamaProvider, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // Ollama can be slower for local models
	}

	proxyFunc := newOllamaProxyFunc(config.HTTPProxy, config.HTTPSProxy, config.NoProxy)

	return &OllamaProvider{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy: proxyFunc,
			},
		},
		config: config,
	}, nil
}

// Name returns the provider name
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// IsAvailable checks if the provider is properly configured
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	// Check if Ollama is running by trying to list models
	url := fmt.Sprintf("%s/api/tags", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ollama availability check failed (request creation): %v\n", err)
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ollama availability check failed (connection to %s): %v\n", p.baseURL, err)
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Ollama availability check failed (HTTP %d from %s)\n", resp.StatusCode, p.baseURL)
		return false
	}

	return true
}

// Summarize generates a summary using Ollama's local models
func (p *OllamaProvider) Summarize(ctx context.Context, req SummarizeRequest) (*SummarizeResponse, error) {
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
		return nil, fmt.Errorf("ollama model must be specified (e.g., llama3.1:8b, mistral)")
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
	apiReq := ollamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false, // Get complete response at once
		System: "You are a helpful assistant that summarizes Entropia reports with strict adherence to evidence constraints.",
		Options: ollamaOptions{
			Temperature: 0.3, // Lower temperature for more focused output
			NumPredict:  maxTokens,
		},
	}

	// Make API call
	resp, err := p.makeRequest(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("ollama API error: %w", err)
	}

	summary := strings.TrimSpace(resp.Response)

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

	// Estimate tokens (Ollama provides counts but they may be 0 for some models)
	tokensUsed := resp.PromptEvalCount + resp.EvalCount
	if tokensUsed == 0 {
		// Rough estimate: 1 token ≈ 4 characters
		tokensUsed = (len(prompt) + len(summary)) / 4
	}

	return &SummarizeResponse{
		Summary:    summary,
		CitedURLs:  citedURLs,
		Model:      resp.Model,
		TokensUsed: tokensUsed,
	}, nil
}

// makeRequest makes an HTTP request to the Ollama API
func (p *OllamaProvider) makeRequest(ctx context.Context, apiReq ollamaRequest) (*ollamaResponse, error) {
	// Serialize request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/generate", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Make request
	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		var apiErr ollamaError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API error (%d): %s", httpResp.StatusCode, apiErr.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var resp ollamaResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}
