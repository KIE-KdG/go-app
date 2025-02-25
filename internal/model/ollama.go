package model

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Models struct {
	Model string
}

func (m *Models) PromptOllama(prompt string) (string, error) {
	llm, err := ollama.New(ollama.WithModel(m.Model))
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	fullPrompt := fmt.Sprintf("Human: %s \nAssistant:", prompt)

	completion, err := llm.Call(ctx, fullPrompt,
		llms.WithTemperature(0.8),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			fmt.Print(string(chunk))
			return nil
		}),
	)
	if err != nil {
		return "", err
	}

	return completion, nil
}

func (m *Models) PromptOllamaStream(prompt string) (<-chan string, error) {
	llm, err := ollama.New(ollama.WithModel(m.Model))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	fullPrompt := fmt.Sprintf("Human: %s \nAssistant:", prompt)

	// Create a channel to send tokens.
	tokenChan := make(chan string)

	go func() {
		// Ensure the channel is closed when done.
		defer close(tokenChan)
		_, err := llm.Call(ctx, fullPrompt,
			llms.WithTemperature(0.8),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				// Send each token (chunk) to the channel.
				tokenChan <- string(chunk)
				return nil
			}),
		)
		if err != nil {
			// Optionally, you can log the error or handle it as needed.
			fmt.Printf("Error during streaming call: %v\n", err)
		}
	}()

	return tokenChan, nil
}