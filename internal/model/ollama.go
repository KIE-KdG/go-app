package model

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Models struct {
}

func (*Models)PromptOllama() (string, error) {
	llm, err := ollama.New(ollama.WithModel("llama3"))
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	completion, err := llm.Call(ctx, "Human: How many r's in strawberry? \nAssistant:",
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