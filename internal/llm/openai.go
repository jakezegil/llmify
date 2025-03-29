package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
	}
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string, model string) (string, error) {
	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant.", // Or more specific system prompt
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			// Add Temperature, MaxTokens etc. from config if desired
		},
	)

	if err != nil {
		return "", fmt.Errorf("openai chat completion error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}
