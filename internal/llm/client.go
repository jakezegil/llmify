package llm

import (
	"context"
	"fmt"

	"github.com/jake/llmify/internal/config" // Use the correct module path
)

// LLMClient defines the interface for interacting with different LLM providers.
type LLMClient interface {
	Generate(ctx context.Context, prompt string, model string) (string, error)
}

// NewLLMClient creates a new LLM client based on the configuration.
func NewLLMClient(cfg *config.Config) (LLMClient, error) {
	apiKey := config.GetAPIKey(cfg.LLM.Provider)

	switch cfg.LLM.Provider {
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not found (set OPENAI_API_KEY or LLMIFY_LLM_API_KEY_OPENAI)")
		}
		return NewOpenAIClient(apiKey), nil
	// case "anthropic":
	//     // ... implementation ...
	// case "ollama":
	// 	   return NewOllamaClient(cfg.LLM.OllamaBaseURL)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLM.Provider)
	}
}
