package forge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/katasec/forge-core/memory/inmem"
)

// mockProvider is a test double that returns pre-configured responses.
type mockProvider struct {
	responses []*ProviderResponse
	errors    []error
	calls     int
}

func (m *mockProvider) Generate(_ context.Context, _ ProviderRequest) (*ProviderResponse, error) {
	i := m.calls
	m.calls++
	if i < len(m.errors) && m.errors[i] != nil {
		return nil, m.errors[i]
	}
	if i < len(m.responses) {
		return m.responses[i], nil
	}
	// Default: stop with empty message.
	return &ProviderResponse{
		Messages:     []Message{AssistantText("default")},
		FinishReason: FinishReasonStop,
	}, nil
}

// recordingProvider stores provider requests so tests can inspect conversation history.
type recordingProvider struct {
	responses []*ProviderResponse
	requests  []ProviderRequest
	calls     int
}

func (r *recordingProvider) Generate(_ context.Context, req ProviderRequest) (*ProviderResponse, error) {
	r.requests = append(r.requests, req)
	i := r.calls
	r.calls++
	if i < len(r.responses) {
		return r.responses[i], nil
	}
	return &ProviderResponse{
		Messages:     []Message{AssistantText("default")},
		FinishReason: FinishReasonStop,
	}, nil
}

func TestNewAgentNilProvider(t *testing.T) {
	_, err := NewAgent(Config{})
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestNewAgentDefaultErrorPolicy(t *testing.T) {
	agent, err := NewAgent(Config{
		Provider: &mockProvider{},
	})
	if err != nil {
		t.Fatalf("NewAgent error: %v", err)
	}
	if agent.errorPolicy != ErrorPolicyStop {
		t.Errorf("errorPolicy = %q, want %q", agent.errorPolicy, ErrorPolicyStop)
	}
}

func TestNewAgentDefaultsToInMemoryStore(t *testing.T) {
	agent, err := NewAgent(Config{
		Provider: &mockProvider{},
	})
	if err != nil {
		t.Fatalf("NewAgent error: %v", err)
	}
	if agent.memory == nil {
		t.Fatal("expected default memory store")
	}
	if _, ok := agent.memory.(*InMemoryStore); !ok {
		t.Fatalf("memory = %T, want *InMemoryStore", agent.memory)
	}
}

func TestNewAgentDisableMemory(t *testing.T) {
	agent, err := NewAgent(Config{
		Provider:      &mockProvider{},
		DisableMemory: true,
	})
	if err != nil {
		t.Fatalf("NewAgent error: %v", err)
	}
	if agent.memory != nil {
		t.Fatalf("memory = %T, want nil", agent.memory)
	}
}

func TestNewAgentAcceptsExplicitMemoryStore(t *testing.T) {
	store := inmem.New()
	agent, err := NewAgent(Config{
		Provider: &mockProvider{},
		Memory:   store,
	})
	if err != nil {
		t.Fatalf("NewAgent error: %v", err)
	}
	if agent.memory != store {
		t.Fatalf("memory = %T, want explicit store", agent.memory)
	}
}

func TestAgentAskPreservesDefaultConversation(t *testing.T) {
	provider := &recordingProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{AssistantText("hello")},
				FinishReason: FinishReasonStop,
			},
			{
				Messages:     []Message{AssistantText("I remember")},
				FinishReason: FinishReasonStop,
			},
		},
	}

	agent, err := NewAgent(Config{Provider: provider})
	if err != nil {
		t.Fatalf("NewAgent error: %v", err)
	}

	first, err := agent.Ask(context.Background(), "My name is Ameer.")
	if err != nil {
		t.Fatalf("Ask first error: %v", err)
	}
	second, err := agent.Ask(context.Background(), "What is my name?")
	if err != nil {
		t.Fatalf("Ask second error: %v", err)
	}

	if first.ConversationID == "" {
		t.Fatal("expected first response to include conversation ID")
	}
	if second.ConversationID != first.ConversationID {
		t.Fatalf("conversation ID = %q, want %q", second.ConversationID, first.ConversationID)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider requests = %d, want 2", len(provider.requests))
	}
	if len(provider.requests[1].Messages) != 3 {
		t.Fatalf("second request messages = %d, want 3", len(provider.requests[1].Messages))
	}
	if provider.requests[1].Messages[0].Text() != "My name is Ameer." {
		t.Errorf("first remembered message = %q", provider.requests[1].Messages[0].Text())
	}
}

func TestAgentAskInUsesNamedConversations(t *testing.T) {
	provider := &recordingProvider{
		responses: []*ProviderResponse{
			{Messages: []Message{AssistantText("forge noted")}, FinishReason: FinishReasonStop},
			{Messages: []Message{AssistantText("other noted")}, FinishReason: FinishReasonStop},
			{Messages: []Message{AssistantText("forge remembered")}, FinishReason: FinishReasonStop},
		},
	}

	agent, err := NewAgent(Config{Provider: provider})
	if err != nil {
		t.Fatalf("NewAgent error: %v", err)
	}

	if _, err := agent.AskIn(context.Background(), "forge", "Remember forge."); err != nil {
		t.Fatalf("AskIn forge error: %v", err)
	}
	if _, err := agent.AskIn(context.Background(), "other", "Remember other."); err != nil {
		t.Fatalf("AskIn other error: %v", err)
	}
	resp, err := agent.AskIn(context.Background(), "forge", "What did I ask you to remember?")
	if err != nil {
		t.Fatalf("AskIn forge follow-up error: %v", err)
	}

	if resp.ConversationID != "forge" {
		t.Fatalf("conversation ID = %q, want forge", resp.ConversationID)
	}
	if len(provider.requests[2].Messages) != 3 {
		t.Fatalf("forge follow-up messages = %d, want 3", len(provider.requests[2].Messages))
	}
	if provider.requests[2].Messages[0].Text() != "Remember forge." {
		t.Errorf("first forge message = %q", provider.requests[2].Messages[0].Text())
	}
}

func TestAgentResponseLastText(t *testing.T) {
	resp := &AgentResponse{
		Messages: []Message{
			UserText("hello"),
			AssistantText("first"),
			{Role: RoleTool, Content: []ContentBlock{ToolResultBlock(ToolResult{Content: "tool result"})}},
			AssistantText("latest"),
		},
	}

	if got := resp.LastText(); got != "latest" {
		t.Fatalf("LastText() = %q, want latest", got)
	}
}

func TestAgentRunStop(t *testing.T) {
	provider := &mockProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{AssistantText("hello back")},
				FinishReason: FinishReasonStop,
				Usage:        TokenUsage{InputTokens: 10, OutputTokens: 5},
			},
		},
	}

	agent, _ := NewAgent(Config{Provider: provider})
	resp, err := agent.Run(context.Background(), AgentRequest{
		Messages: []Message{UserText("hello")},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if resp.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, FinishReasonStop)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Errorf("Usage = %+v", resp.Usage)
	}
	if len(resp.Messages) != 2 { // user + assistant
		t.Errorf("got %d messages, want 2", len(resp.Messages))
	}
	if resp.ConversationID == "" {
		t.Error("expected ConversationID to be generated")
	}
}

func TestAgentRunIterLimit(t *testing.T) {
	provider := &mockProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{{Role: RoleAssistant, Content: []ContentBlock{ToolCallBlock(ToolCall{ID: "c1", Name: "echo", Arguments: json.RawMessage(`{"text":"hi"}`)})}}},
				FinishReason: FinishReasonToolUse,
			},
			// Would loop forever, but iter limit stops it.
			{
				Messages:     []Message{{Role: RoleAssistant, Content: []ContentBlock{ToolCallBlock(ToolCall{ID: "c2", Name: "echo", Arguments: json.RawMessage(`{"text":"hi"}`)})}}},
				FinishReason: FinishReasonToolUse,
			},
		},
	}

	type echoInput struct {
		Text string `json:"text"`
	}

	agent, _ := NewAgent(Config{
		Provider:      provider,
		MaxIterations: 2,
		Tools: []Tool{
			Func[echoInput]("echo", "echoes", func(_ context.Context, in echoInput) (string, error) {
				return in.Text, nil
			}),
		},
	})

	resp, err := agent.Run(context.Background(), AgentRequest{
		Messages: []Message{UserText("go")},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.FinishReason != FinishReasonIterLimit {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, FinishReasonIterLimit)
	}
}

func TestAgentRunProviderError(t *testing.T) {
	provider := &mockProvider{
		errors: []error{errors.New("provider down")},
	}

	agent, _ := NewAgent(Config{Provider: provider})
	_, err := agent.Run(context.Background(), AgentRequest{
		Messages: []Message{UserText("hello")},
	})
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if err.Error() != "provider down" {
		t.Errorf("error = %q, want %q", err.Error(), "provider down")
	}
}

func TestAgentRunToolErrorStop(t *testing.T) {
	provider := &mockProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{{Role: RoleAssistant, Content: []ContentBlock{ToolCallBlock(ToolCall{ID: "c1", Name: "broken", Arguments: json.RawMessage(`{}`)})}}},
				FinishReason: FinishReasonToolUse,
			},
		},
	}

	type emptyInput struct{}

	agent, _ := NewAgent(Config{
		Provider:    provider,
		ErrorPolicy: ErrorPolicyStop,
		Tools: []Tool{
			Func[emptyInput]("broken", "always fails", func(_ context.Context, _ emptyInput) (string, error) {
				return "", errors.New("tool broke")
			}),
		},
	})

	resp, err := agent.Run(context.Background(), AgentRequest{
		Messages: []Message{UserText("go")},
	})
	if err != nil {
		t.Fatalf("Run error: %v (tool errors should not be fatal)", err)
	}
	if resp.FinishReason != FinishReasonError {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, FinishReasonError)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("got %d errors, want 1", len(resp.Errors))
	}
	if resp.Errors[0].Message != "tool broke" {
		t.Errorf("error message = %q", resp.Errors[0].Message)
	}
	// Tool results should still be in the message history.
	lastMsg := resp.Messages[len(resp.Messages)-1]
	if lastMsg.Role != RoleTool {
		t.Errorf("last message role = %q, want %q", lastMsg.Role, RoleTool)
	}
}

func TestAgentRunToolErrorContinue(t *testing.T) {
	type emptyInput struct{}

	provider := &mockProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{{Role: RoleAssistant, Content: []ContentBlock{ToolCallBlock(ToolCall{ID: "c1", Name: "broken", Arguments: json.RawMessage(`{}`)})}}},
				FinishReason: FinishReasonToolUse,
			},
			// After seeing the error, LLM stops.
			{
				Messages:     []Message{AssistantText("I see the tool failed")},
				FinishReason: FinishReasonStop,
			},
		},
	}

	agent, _ := NewAgent(Config{
		Provider:    provider,
		ErrorPolicy: ErrorPolicyContinue,
		Tools: []Tool{
			Func[emptyInput]("broken", "always fails", func(_ context.Context, _ emptyInput) (string, error) {
				return "", errors.New("tool broke")
			}),
		},
	})

	resp, err := agent.Run(context.Background(), AgentRequest{
		Messages: []Message{UserText("go")},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.FinishReason != FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q (should continue past error)", resp.FinishReason, FinishReasonStop)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("got %d errors, want 1 (errors should still be collected)", len(resp.Errors))
	}
	if provider.calls != 2 {
		t.Errorf("provider called %d times, want 2", provider.calls)
	}
}

func TestAgentRunWithMemory(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Pre-populate memory.
	store.Save(ctx, "conv-1", []Message{
		UserText("earlier message"),
	})

	provider := &mockProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{AssistantText("I remember")},
				FinishReason: FinishReasonStop,
				Usage:        TokenUsage{InputTokens: 20, OutputTokens: 10},
			},
		},
	}

	agent, _ := NewAgent(Config{
		Provider: provider,
		Memory:   store,
	})

	resp, err := agent.Run(ctx, AgentRequest{
		ConversationID: "conv-1",
		Messages:       []Message{UserText("new message")},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// Should have: earlier + new + assistant = 3 messages.
	if len(resp.Messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(resp.Messages))
	}
	if resp.Messages[0].Text() != "earlier message" {
		t.Errorf("Messages[0] = %q, want %q", resp.Messages[0].Text(), "earlier message")
	}

	// Memory should be updated with all 3 messages.
	saved, _ := store.Load(ctx, "conv-1")
	if len(saved) != 3 {
		t.Fatalf("saved %d messages, want 3", len(saved))
	}
}

func TestAgentRunUsageAccumulation(t *testing.T) {
	type emptyInput struct{}

	provider := &mockProvider{
		responses: []*ProviderResponse{
			{
				Messages:     []Message{{Role: RoleAssistant, Content: []ContentBlock{ToolCallBlock(ToolCall{ID: "c1", Name: "noop", Arguments: json.RawMessage(`{}`)})}}},
				FinishReason: FinishReasonToolUse,
				Usage:        TokenUsage{InputTokens: 10, OutputTokens: 5},
			},
			{
				Messages:     []Message{AssistantText("done")},
				FinishReason: FinishReasonStop,
				Usage:        TokenUsage{InputTokens: 20, OutputTokens: 8},
			},
		},
	}

	agent, _ := NewAgent(Config{
		Provider: provider,
		Tools: []Tool{
			Func[emptyInput]("noop", "does nothing", func(_ context.Context, _ emptyInput) (string, error) {
				return "ok", nil
			}),
		},
	})

	resp, err := agent.Run(context.Background(), AgentRequest{
		Messages: []Message{UserText("go")},
	})
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Usage.InputTokens != 30 {
		t.Errorf("InputTokens = %d, want 30", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 13 {
		t.Errorf("OutputTokens = %d, want 13", resp.Usage.OutputTokens)
	}
}
