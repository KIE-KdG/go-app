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