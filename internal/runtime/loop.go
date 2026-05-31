package runtime

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/provider"
	"github.com/katasec/forge-core/tool"
)

// Run executes the agent loop.
func (a *Agent) Run(ctx context.Context, req AgentRequest) (*AgentResponse, error) {
	state, err := a.startRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := a.runIterations(ctx, state); err != nil {
		return nil, err
	}

	if err := a.saveMessages(ctx, state); err != nil {
		return nil, err
	}

	return state.response(), nil
}

type runState struct {
	conversationID string
	messages       []message.Message
	usage          provider.TokenUsage
	toolErrors     []tool.Error
	finishReason   provider.FinishReason
	iteration      int
}

func (a *Agent) startRun(ctx context.Context, req AgentRequest) (*runState, error) {
	conversationID := resolveConversationID(req.ConversationID)
	messages, err := a.loadMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	return &runState{
		conversationID: conversationID,
		messages:       append(messages, req.Messages...),
	}, nil
}

func resolveConversationID(conversationID string) string {
	if conversationID != "" {
		return conversationID
	}
	return uuid.New().String()
}

func (a *Agent) loadMessages(ctx context.Context, conversationID string) ([]message.Message, error) {
	if a.memory == nil {
		return nil, nil
	}
	return a.memory.Load(ctx, conversationID)
}

func (a *Agent) runIterations(ctx context.Context, state *runState) error {
	for {
		if a.reachedIterationLimit(state) {
			state.finishReason = provider.FinishReasonIterLimit
			return nil
		}

		resp, err := a.callProvider(ctx, state.messages)
		if err != nil {
			return err
		}
		state.recordProviderResponse(resp)

		if resp.FinishReason == provider.FinishReasonStop {
			state.finishReason = provider.FinishReasonStop
			return nil
		}

		if a.handleToolUse(ctx, state, resp) {
			return nil
		}
	}
}

func (a *Agent) reachedIterationLimit(state *runState) bool {
	return a.maxIterations > 0 && state.iteration >= a.maxIterations
}

func (a *Agent) callProvider(ctx context.Context, messages []message.Message) (*provider.Response, error) {
	return a.run(ctx, provider.Request{
		Messages:     messages,
		Tools:        a.registry.Definitions(),
		SystemPrompt: a.systemPrompt,
	})
}

func (s *runState) recordProviderResponse(resp *provider.Response) {
	s.usage = addUsage(s.usage, resp.Usage)
	s.messages = append(s.messages, resp.Messages...)
	s.iteration++
}

func (a *Agent) handleToolUse(ctx context.Context, state *runState, resp *provider.Response) bool {
	calls, err := lastToolCalls(resp.Messages)
	if err != nil {
		state.finishReason = provider.FinishReasonError
		state.toolErrors = append(state.toolErrors, tool.Error{Message: err.Error()})
		return true
	}

	results := a.executor.Execute(ctx, calls)
	state.toolErrors = append(state.toolErrors, collectToolErrors(results)...)
	state.messages = append(state.messages, message.ToolMessage(results...))

	if a.shouldStopAfterToolResults(results) {
		state.finishReason = provider.FinishReasonError
		return true
	}

	return false
}

func lastToolCalls(messages []message.Message) ([]tool.Call, error) {
	if len(messages) == 0 {
		return nil, errors.New("provider requested tool use without a message")
	}
	calls := messages[len(messages)-1].ToolCalls()
	if len(calls) == 0 {
		return nil, errors.New("provider requested tool use without tool calls")
	}
	return calls, nil
}

func collectToolErrors(results []tool.Result) []tool.Error {
	var errors []tool.Error
	for _, result := range results {
		if result.IsError {
			errors = append(errors, tool.Error{
				CallID:  result.CallID,
				Message: result.Content,
			})
		}
	}
	return errors
}

func (a *Agent) shouldStopAfterToolResults(results []tool.Result) bool {
	return a.errorPolicy == ErrorPolicyStop && hasToolError(results)
}

func hasToolError(results []tool.Result) bool {
	for _, result := range results {
		if result.IsError {
			return true
		}
	}
	return false
}

func (a *Agent) saveMessages(ctx context.Context, state *runState) error {
	if a.memory == nil {
		return nil
	}
	return a.memory.Save(ctx, state.conversationID, state.messages)
}

func (s *runState) response() *AgentResponse {
	return &AgentResponse{
		ConversationID: s.conversationID,
		Messages:       s.messages,
		FinishReason:   s.finishReason,
		Usage:          s.usage,
		Errors:         s.toolErrors,
	}
}

func addUsage(a, b provider.TokenUsage) provider.TokenUsage {
	return provider.TokenUsage{
		InputTokens:           a.InputTokens + b.InputTokens,
		CachedInputTokens:     a.CachedInputTokens + b.CachedInputTokens,
		OutputTokens:          a.OutputTokens + b.OutputTokens,
		ReasoningOutputTokens: a.ReasoningOutputTokens + b.ReasoningOutputTokens,
		TotalTokens:           a.TotalTokens + b.TotalTokens,
	}
}
