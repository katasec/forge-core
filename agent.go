package forge

import (
	"context"

	"github.com/katasec/forge-core/internal/runtime"
)

// Agent orchestrates the LLM call -> tool execution -> response loop.
type Agent struct {
	inner *runtime.Agent
}

// NewAgent creates an Agent from the given Config.
func NewAgent(cfg Config) (*Agent, error) {
	agent, err := runtime.NewAgent(cfg)
	if err != nil {
		return nil, err
	}
	return &Agent{inner: agent}, nil
}

// Ask sends a user prompt in the agent's default conversation.
func (a *Agent) Ask(ctx context.Context, prompt string) (*AgentResponse, error) {
	return a.inner.Ask(ctx, prompt)
}

// AskIn sends a user prompt in the named conversation.
func (a *Agent) AskIn(ctx context.Context, conversationID, prompt string) (*AgentResponse, error) {
	return a.inner.AskIn(ctx, conversationID, prompt)
}

// AskContent sends a rich user message in the agent's default conversation.
func (a *Agent) AskContent(ctx context.Context, blocks ...ContentBlock) (*AgentResponse, error) {
	return a.inner.AskContent(ctx, blocks...)
}

// Run executes the agent loop.
func (a *Agent) Run(ctx context.Context, req AgentRequest) (*AgentResponse, error) {
	return a.inner.Run(ctx, req)
}
