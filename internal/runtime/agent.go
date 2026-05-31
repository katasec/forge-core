package runtime

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/katasec/forge-core/executor"
	"github.com/katasec/forge-core/executor/sequential"
	"github.com/katasec/forge-core/memory"
	"github.com/katasec/forge-core/memory/inmem"
	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/middleware"
	"github.com/katasec/forge-core/provider"
	"github.com/katasec/forge-core/tool"
	"github.com/katasec/forge-core/tool/registry"
)

// Agent orchestrates the LLM call -> tool execution -> response loop.
type Agent struct {
	provider              provider.Provider
	registry              *registry.Registry
	executor              executor.Executor
	run                   middleware.RunFunc
	memory                memory.Store
	defaultConversationID string
	systemPrompt          string
	maxIterations         int
	errorPolicy           ErrorPolicy
}

// NewAgent creates an Agent from the given Config.
func NewAgent(cfg Config) (*Agent, error) {
	if cfg.Provider == nil {
		return nil, errors.New("forge: provider must not be nil")
	}

	registry := buildRegistry(cfg.Tools)

	return &Agent{
		provider:              cfg.Provider,
		registry:              registry,
		executor:              buildExecutor(registry),
		run:                   composeRunFunc(cfg.Provider, cfg.Middleware),
		memory:                selectMemory(cfg.Memory, cfg.DisableMemory),
		defaultConversationID: uuid.New().String(),
		systemPrompt:          cfg.SystemPrompt,
		maxIterations:         cfg.MaxIterations,
		errorPolicy:           selectErrorPolicy(cfg.ErrorPolicy),
	}, nil
}

// Ask sends a user prompt in the agent's default conversation.
func (a *Agent) Ask(ctx context.Context, prompt string) (*AgentResponse, error) {
	return a.AskIn(ctx, a.defaultConversationID, prompt)
}

// AskIn sends a user prompt in the named conversation.
func (a *Agent) AskIn(ctx context.Context, conversationID, prompt string) (*AgentResponse, error) {
	return a.Run(ctx, AgentRequest{
		ConversationID: conversationID,
		Messages:       []message.Message{message.UserText(prompt)},
	})
}

// AskContent sends a rich user message in the agent's default conversation.
func (a *Agent) AskContent(ctx context.Context, blocks ...message.ContentBlock) (*AgentResponse, error) {
	return a.Run(ctx, AgentRequest{
		ConversationID: a.defaultConversationID,
		Messages:       []message.Message{message.UserMessage(blocks...)},
	})
}

func buildRegistry(tools []tool.Tool) *registry.Registry {
	r := registry.New()
	if len(tools) > 0 {
		r.Register(tools...)
	}
	return r
}

func buildExecutor(registry *registry.Registry) executor.Executor {
	return &sequential.Executor{Registry: registry}
}

func composeRunFunc(p provider.Provider, middlewares []middleware.Middleware) middleware.RunFunc {
	run := middleware.RunFunc(p.Generate)
	for i := len(middlewares) - 1; i >= 0; i-- {
		run = middlewares[i](run)
	}
	return run
}

func selectMemory(store memory.Store, disabled bool) memory.Store {
	if store != nil || disabled {
		return store
	}
	return inmem.New()
}

func selectErrorPolicy(policy ErrorPolicy) ErrorPolicy {
	if policy == "" {
		return ErrorPolicyStop
	}
	return policy
}
