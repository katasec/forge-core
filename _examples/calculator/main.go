// Calculator demonstrates kiln with math tools and a mock provider.
//
// The mock provider simulates an LLM deciding to call a tool, so the whole
// agent loop runs with no API key. Swap in a real provider (anthropic, openai,
// xai) to talk to an actual model.
//
// Run: go run .
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/katasec/kiln"
	"github.com/katasec/kiln/message"
	"github.com/katasec/kiln/tool"
)

// --- Tool input types ---

type AddInput struct {
	A float64 `json:"a" jsonschema:"description=First number"`
	B float64 `json:"b" jsonschema:"description=Second number"`
}

type MultiplyInput struct {
	A float64 `json:"a" jsonschema:"description=First number"`
	B float64 `json:"b" jsonschema:"description=Second number"`
}

// --- Mock provider ---

// MockProvider simulates an LLM that uses calculator tools. On the first call
// it requests tool use; on the second it returns the final answer.
type MockProvider struct {
	calls int
}

func (p *MockProvider) Generate(_ context.Context, req kiln.ProviderRequest) (*kiln.ProviderResponse, error) {
	p.calls++

	// First call: the "LLM" decides to use the add tool.
	if p.calls == 1 {
		return &kiln.ProviderResponse{
			Messages: []kiln.Message{{
				Role: kiln.RoleAssistant,
				Content: []kiln.ContentBlock{
					message.ToolCall(kiln.ToolCall{
						ID:        "call-1",
						Name:      "add",
						Arguments: json.RawMessage(`{"a": 12, "b": 30}`),
					}),
				},
			}},
			FinishReason: kiln.FinishReasonToolUse,
			Usage:        kiln.TokenUsage{InputTokens: 25, OutputTokens: 15},
		}, nil
	}

	// Second call: the "LLM" sees the tool result and formulates the answer.
	var toolResult string
	for _, msg := range req.Messages {
		if msg.Role == kiln.RoleTool && len(msg.ToolResults()) > 0 {
			toolResult = msg.ToolResults()[0].Content
		}
	}

	return &kiln.ProviderResponse{
		Messages:     []kiln.Message{message.AssistantText(fmt.Sprintf("The answer is %s!", toolResult))},
		FinishReason: kiln.FinishReasonStop,
		Usage:        kiln.TokenUsage{InputTokens: 40, OutputTokens: 10},
	}, nil
}

func main() {
	// Create tools. The JSON schema is derived from the input struct.
	addTool := tool.Func[AddInput, float64]("add", "Add two numbers",
		func(_ context.Context, in AddInput) (float64, error) {
			return in.A + in.B, nil
		})

	mulTool := tool.Func[MultiplyInput, float64]("multiply", "Multiply two numbers",
		func(_ context.Context, in MultiplyInput) (float64, error) {
			return in.A * in.B, nil
		})

	// Log every provider call.
	logging := kiln.Middleware(func(next kiln.RunFunc) kiln.RunFunc {
		return func(ctx context.Context, req kiln.ProviderRequest) (*kiln.ProviderResponse, error) {
			fmt.Printf("[middleware] Calling provider with %d messages\n", len(req.Messages))
			resp, err := next(ctx, req)
			if err == nil {
				fmt.Printf("[middleware] Provider returned: finish_reason=%s\n", resp.FinishReason)
			}
			return resp, err
		}
	})

	// Build the agent once: provider, tools, middleware, and loop policy.
	agent, err := kiln.NewAgent(kiln.Config{
		Provider:      &MockProvider{},
		Tools:         []kiln.Tool{addTool, mulTool},
		Middleware:    []kiln.Middleware{logging},
		SystemPrompt:  "You are a helpful calculator assistant.",
		MaxIterations: 5,
		ErrorPolicy:   kiln.ErrorPolicyContinue,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Ask drives the full provider -> tool -> provider loop.
	fmt.Println("User: What is 12 + 30?")
	fmt.Println(strings.Repeat("-", 40))

	resp, err := agent.Ask(context.Background(), "What is 12 + 30?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Assistant: %s\n", resp.LastText())
	fmt.Printf("Finish reason: %s\n", resp.FinishReason)
	fmt.Printf("Tokens: %d in, %d out\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
	fmt.Printf("Conversation: %d messages\n", len(resp.Messages))
}
