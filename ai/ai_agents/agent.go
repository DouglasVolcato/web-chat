package ai_agents

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

type ToolResult struct {
	Response           string
	ReturnValueToAgent bool
}

type Tool interface {
	GetDefinition() openai.FunctionDefinition
	Handle(functionCall string, arguments []byte) (ToolResult, error)
}

type Agent struct {
	prompt string
	tools  []Tool
}

func NewAgent(prompt string, tools []Tool) Agent {
	return Agent{prompt: prompt, tools: tools}
}

func (a *Agent) Answer(ctx context.Context, questions []openai.ChatCompletionMessage) (string, error) {
	config := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))
	config.HTTPClient = &http.Client{
		Timeout: 300 * time.Second,
	}
	client := openai.NewClientWithConfig(config)

	toolDefinitions := []openai.FunctionDefinition{}
	for _, tool := range a.tools {
		toolDefinitions = append(toolDefinitions, tool.GetDefinition())
	}

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     openai.GPT4oMini,
		Messages:  append([]openai.ChatCompletionMessage{{Role: "system", Content: a.prompt}}, questions...),
		Functions: toolDefinitions,
	})

	if err != nil {
		return "", err
	}

	message := resp.Choices[0].Message

	if message.FunctionCall != nil {
		for _, tool := range a.tools {
			if tool.GetDefinition().Name == message.FunctionCall.Name {
				result, err := tool.Handle(message.FunctionCall.Name, []byte(message.FunctionCall.Arguments))
				if err != nil {
					return "", err
				}

				if result.ReturnValueToAgent {
					return a.Answer(ctx, append(questions, openai.ChatCompletionMessage{Role: "function", Name: message.FunctionCall.Name, Content: result.Response}))
				} else {
					return result.Response, nil
				}
			}
		}
	}

	return message.Content, nil
}
