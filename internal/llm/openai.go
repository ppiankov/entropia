package llm

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI models
type OpenAIProvider struct {
	client *openai.Client
	config Config
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config Config) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	var clientConfig openai.ClientConfig
	if config.BaseURL != "" {
		clientConfig = openai.DefaultConfig(config.APIKey)
		clientConfig.BaseURL = config.BaseURL
	} else {
		clientConfig = openai.DefaultConfig(config.APIKey)
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// IsAvailable checks if the provider is properly configured
func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool {
	// Simple check: try to list models (lightweight API call)
	_, err := p.client.ListModels(ctx)
	if err != nil {
		// Log the actual error for debugging (this helps users diagnose API key issues)
		fmt.Fprintf(os.Stderr, "OpenAI API check failed: %v\n", err)
		return false
	}
	return true
}

// Summarize generates a summary using OpenAI's Chat Completions API
func (p *OpenAIProvider) Summarize(ctx context.Context, req SummarizeRequest) (*SummarizeResponse, error) {
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
		model = openai.GPT4oMini // Default to gpt-4o-mini
	}

	// Determine max tokens
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens
	}
	if maxTokens == 0 {
		maxTokens = 1000
	}

	// Create timeout context
	timeout := time.Duration(p.config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Make API call
	chatReq := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant that summarizes Entropia reports with strict adherence to evidence constraints.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.3, // Lower temperature for more focused, factual output
	}

	resp, err := p.client.CreateChatCompletion(ctxWithTimeout, chatReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)

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

	return &SummarizeResponse{
		Summary:    summary,
		CitedURLs:  citedURLs,
		Model:      model,
		TokensUsed: resp.Usage.TotalTokens,
	}, nil
}

// extractURLs extracts all URLs from text using regex
func extractURLs(text string) []string {
	// Match http(s) URLs
	urlPattern := regexp.MustCompile(`https?://[^\s\)]+`)
	matches := urlPattern.FindAllString(text, -1)

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, url := range matches {
		// Clean up trailing punctuation
		url = strings.TrimRight(url, ".,;:!?")
		if !seen[url] {
			seen[url] = true
			unique = append(unique, url)
		}
	}

	return unique
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
