package llm

import (
	"fmt"
	"strings"

	"github.com/ppiankov/entropia/internal/model"
)

// NewProvider creates a new LLM provider based on configuration
func NewProvider(config Config) (Provider, error) {
	provider := strings.ToLower(config.Provider)

	switch provider {
	case "openai":
		return NewOpenAIProvider(config)

	case "anthropic", "claude":
		return NewAnthropicProvider(config)

	case "ollama":
		return NewOllamaProvider(config)

	case "":
		// No provider configured - return nil (LLM disabled)
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s (supported: openai, anthropic, ollama)", config.Provider)
	}
}

// ConfigFromModel converts model.LLMConfig to llm.Config
func ConfigFromModel(modelConfig model.LLMConfig) Config {
	return Config{
		Provider:       modelConfig.Provider,
		Model:          modelConfig.Model,
		APIKey:         modelConfig.APIKey,
		BaseURL:        modelConfig.BaseURL,
		Timeout:        modelConfig.Timeout,
		StrictEvidence: modelConfig.StrictEvidence,
		MaxTokens:      modelConfig.MaxTokens,
	}
}

// LoadConfigFromEnv loads LLM configuration from environment variables
func LoadConfigFromEnv() Config {
	// This will be expanded to read from env vars
	// For now, return defaults
	return DefaultConfig()
}
