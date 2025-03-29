package llm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	// Create a custom HTTP client with longer timeouts
	httpClient := &http.Client{
		Timeout: 180 * time.Second, // 3 minute timeout for HTTP requests
	}

	config := openai.DefaultConfig(apiKey)
	config.HTTPClient = httpClient

	return &OpenAIClient{
		client: openai.NewClientWithConfig(config),
	}
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string, model string) (string, error) {
	verbose := viper.GetBool("verbose")

	// Use a fallback model if the model is not specified
	if model == "" {
		model = "gpt-3.5-turbo" // Default model
		if verbose {
			log.Printf("No model specified, using default model: %s", model)
		}
	}

	// Create request with more conservative settings for stability
	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant specialized in refactoring code. Provide complete refactored code without explanations.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature:      0.2,  // Lower temperature for more deterministic output
		MaxTokens:        4096, // Higher limit for larger code bases
		TopP:             0.95, // More focused sampling
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}

	// Try with exponential backoff
	maxRetries := 3
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Add delay with exponential backoff for retries
		if attempt > 0 {
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			if verbose {
				log.Printf("Retrying OpenAI request (attempt %d/%d) after %v delay",
					attempt+1, maxRetries, backoffDuration)
			}

			// Wait with context awareness
			select {
			case <-time.After(backoffDuration):
				// Waited successfully
			case <-ctx.Done():
				return "", fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
			}
		}

		// Make the API call
		resp, err := c.client.CreateChatCompletion(ctx, req)

		// If successful, return the result
		if err == nil {
			if len(resp.Choices) == 0 {
				return "", fmt.Errorf("OpenAI returned no choices")
			}
			return resp.Choices[0].Message.Content, nil
		}

		// Check if we should retry based on the type of error
		lastError = err
		if ctx.Err() != nil {
			// Don't retry if context was cancelled
			if verbose {
				log.Printf("Context cancelled or timed out, not retrying: %v", ctx.Err())
			}
			break
		}

		if verbose {
			log.Printf("OpenAI API error (attempt %d/%d): %v", attempt+1, maxRetries, err)
		}
	}

	return "", fmt.Errorf("OpenAI chat completion failed after %d attempts: %w", maxRetries, lastError)
}
