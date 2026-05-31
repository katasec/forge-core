package forge

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/katasec/forge-core/message"
)

// AgentRequest is the input to Agent.Run.
type AgentRequest struct {
	ConversationID string    `json:"conversation_id"`
	Messages       []Message `json:"messages"`
}

// AgentResponse is the output of Agent.Run.
type AgentResponse struct {
	ConversationID string       `json:"conversation_id"`
	Messages       []Message    `json:"messages"`
	FinishReason   FinishReason `json:"finish_reason"`
	Usage          TokenUsage   `json:"usage"`
	Errors         []ToolError  `json:"errors,omitempty"`
}

// LastText returns the latest assistant text in the response.
func (r *AgentResponse) LastText() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		msg := r.Messages[i]
		if msg.Role == RoleAssistant && msg.Text() != "" {
			return msg.Text()
		}
	}
	return ""
}

// Agent orchestrates the LLM call -> tool execution -> response loop.
type Agent struct {
	provider              Provider
	registry              *ToolRegistry
	executor              ToolExecutor
	run                   RunFunc
	memory                MemoryStore
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

	registry := NewToolRegistry()
	if len(cfg.Tools) > 0 {
		registry.Register(cfg.Tools...)
	}

	executor := &SequentialExecutor{Registry: registry}

	// Build composed RunFunc from provider + middleware.
	run := RunFunc(cfg.Provider.Generate)
	for i := len(cfg.Middleware) - 1; i >= 0; i-- {
		run = cfg.Middleware[i](run)
	}

	errorPolicy := cfg.ErrorPolicy
	if errorPolicy == "" {
		errorPolicy = ErrorPolicyStop
	}

	memory := cfg.Memory
	if memory == nil && !cfg.DisableMemory {
		memory = NewInMemoryStore()
	}

	return &Agent{
		provider:              cfg.Provider,
		registry:              registry,
		executor:              executor,
		run:                   run,
		memory:                memory,
		defaultConversationID: uuid.New().String(),
		systemPrompt:          cfg.SystemPrompt,
		maxIterations:         cfg.MaxIterations,
		errorPolicy:           errorPolicy,
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
		Messages:       []Message{UserText(prompt)},
	})
}

// AskContent sends a rich user message in the agent's default conversation.
func (a *Agent) AskContent(ctx context.Context, blocks ...ContentBlock) (*AgentResponse, error) {
	return a.Run(ctx, AgentRequest{
		ConversationID: a.defaultConversationID,
		Messages:       []Message{UserMessage(blocks...)},
	})
}

// Run executes the agent loop per the design spec pseudocode.
func (a *Agent) Run(ctx context.Context, req AgentRequest) (*AgentResponse, error) {
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = uuid.New().String()
	}

	var messages []Message

	// Load from memory if configured.
	if a.memory != nil {
		loaded, err := a.memory.Load(ctx, conversationID)
		if err != nil {
			return nil, err
		}
		messages = loaded
	}

	messages = append(messages, req.Messages...)

	var (
		usage        TokenUsage
		toolErrors   []ToolError
		finishReason FinishReason
		iteration    int
	)

	for {
		// Check iteration limit.
		if a.maxIterations > 0 && iteration >= a.maxIterations {
			finishReason = FinishReasonIterLimit
			break
		}

		providerReq := ProviderRequest{
			Messages:     messages,
			Tools:        a.registry.Definitions(),
			SystemPrompt: a.systemPrompt,
		}

		resp, err := a.run(ctx, providerReq)
		if err != nil {
			return nil, err // provider errors are always fatal
		}

		usage.InputTokens += resp.Usage.InputTokens
		usage.CachedInputTokens += resp.Usage.CachedInputTokens
		usage.OutputTokens += resp.Usage.OutputTokens
		usage.ReasoningOutputTokens += resp.Usage.ReasoningOutputTokens
		usage.TotalTokens += resp.Usage.TotalTokens
		messages = append(messages, resp.Messages...)
		iteration++

		if resp.FinishReason == FinishReasonStop {
			finishReason = FinishReasonStop
			break
		}

		// FinishReason is tool_use - execute the tool calls.
		if len(resp.Messages) == 0 {
			finishReason = FinishReasonError
			toolErrors = append(toolErrors, ToolError{Message: "provider requested tool use without a message"})
			break
		}
		toolCalls := resp.Messages[len(resp.Messages)-1].ToolCalls()
		if len(toolCalls) == 0 {
			finishReason = FinishReasonError
			toolErrors = append(toolErrors, ToolError{Message: "provider requested tool use without tool calls"})
			break
		}
		toolResults := a.executor.Execute(ctx, toolCalls)

		// Check for tool errors.
		hasError := false
		for _, result := range toolResults {
			if result.IsError {
				toolErrors = append(toolErrors, ToolError{
					CallID:  result.CallID,
					Message: result.Content,
				})
				if a.errorPolicy == ErrorPolicyStop {
					finishReason = FinishReasonError
					hasError = true
					break
				}
			}
		}

		// Append tool results message (even on error, for coherent history).
		messages = append(messages, message.ToolMessage(toolResults...))

		if hasError {
			break
		}
	}

	// Save to memory if configured.
	if a.memory != nil {
		if err := a.memory.Save(ctx, conversationID, messages); err != nil {
			return nil, err
		}
	}

	return &AgentResponse{
		ConversationID: conversationID,
		Messages:       messages,
		FinishReason:   finishReason,
		Usage:          usage,
		Errors:         toolErrors,
	}, nil
}
