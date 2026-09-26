package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/katasec/forge-core"
	"github.com/katasec/forge-core/message"
)

// Compile-time check that *AnthropicProvider satisfies forge.Provider.
var _ forge.Provider = (*AnthropicProvider)(nil)

func TestNew(t *testing.T) {
	p := New("test-key", ModelClaudeSonnet5)
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", p.apiKey, "test-key")
	}
	if p.model != ModelClaudeSonnet5 {
		t.Errorf("model = %q, want %q", p.model, ModelClaudeSonnet5)
	}
}

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request shape.
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q, want %q", got, "test-key")
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Errorf("anthropic-version = %q, want %q", got, "2023-06-01")
		}

		var req apiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "claude-sonnet-5" {
			t.Errorf("model = %q, want %q", req.Model, "claude-sonnet-5")
		}
		if len(req.System) != 1 || req.System[0].Text != "You are helpful." {
			t.Errorf("system = %q, want %q", req.System, "You are helpful.")
		}

		resp := apiResponse{
			ID:         "msg_test",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-sonnet-5",
			Content:    []contentBlock{{Type: "text", Text: "Hello!"}},
			StopReason: "end_turn",
			Usage:      usageBlock{InputTokens: 10, OutputTokens: 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("test-key", ModelClaudeSonnet5, WithBaseURL(srv.URL))

	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		SystemPrompt: "You are helpful.",
		Messages: []forge.Message{
			message.UserText("Hi"),
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.Messages[0].Text() != "Hello!" {
		t.Errorf("content = %q, want %q", resp.Messages[0].Text(), "Hello!")
	}
	if resp.Messages[0].Role != forge.RoleAssistant {
		t.Errorf("role = %q, want %q", resp.Messages[0].Role, forge.RoleAssistant)
	}
	if resp.FinishReason != forge.FinishReasonStop {
		t.Errorf("finishReason = %q, want %q", resp.FinishReason, forge.FinishReasonStop)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Errorf("usage = %+v, want {10, 5}", resp.Usage)
	}
}

func TestGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer srv.Close()

	p := New("bad-key", ModelClaudeSonnet5, WithBaseURL(srv.URL))

	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{message.UserText("Hi")},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

type apiRequest struct {
	Model  string         `json:"model"`
	System []contentBlock `json:"system"`
}

type apiResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Model      string         `json:"model"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      usageBlock     `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usageBlock struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// TestGenerateSendsToolsAndParsesToolUse covers the full tool round trip: the
// definitions reach the wire, and a tool_use response becomes a Forge ToolCall.
func TestGenerateSendsToolsAndParsesToolUse(t *testing.T) {
	var got toolRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_test", "type": "message", "role": "assistant",
			"model": "claude-sonnet-5", "stop_reason": "tool_use",
			"content": []map[string]any{
				{"type": "tool_use", "id": "toolu_1", "name": "search", "input": map[string]any{"query": "go"}},
			},
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelClaudeSonnet5, WithBaseURL(srv.URL))
	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{message.UserText("find something")},
		Tools: []forge.ToolDefinition{{
			Name:        "search",
			Description: "Search the database",
			Schema:      forge.ToolSchema{Parameters: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`)},
		}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(got.Tools) != 1 {
		t.Fatalf("tools sent = %d, want 1", len(got.Tools))
	}
	if got.Tools[0].Name != "search" || got.Tools[0].Description != "Search the database" {
		t.Errorf("tool = %+v, want name/description to survive", got.Tools[0])
	}
	if _, ok := got.Tools[0].InputSchema.Properties["query"]; !ok {
		t.Errorf("input_schema properties = %v, want 'query'", got.Tools[0].InputSchema.Properties)
	}

	if resp.FinishReason != forge.FinishReasonToolUse {
		t.Errorf("finish reason = %q, want %q", resp.FinishReason, forge.FinishReasonToolUse)
	}
	calls := resp.Messages[0].ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(calls))
	}
	if calls[0].ID != "toolu_1" || calls[0].Name != "search" {
		t.Errorf("call = %+v, want id toolu_1 name search", calls[0])
	}
	if string(calls[0].Arguments) != `{"query":"go"}` {
		t.Errorf("arguments = %s, want {\"query\":\"go\"}", calls[0].Arguments)
	}
}

// TestGenerateSendsToolResults checks tool results are replayed as a user
// message carrying tool_result blocks, which is the shape Anthropic requires.
func TestGenerateSendsToolResults(t *testing.T) {
	var got toolRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "m", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"stop_reason": "end_turn",
			"content":     []map[string]any{{"type": "text", "text": "done"}},
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelClaudeSonnet5, WithBaseURL(srv.URL))
	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{
			message.UserText("find something"),
			{Role: forge.RoleAssistant, Content: []forge.ContentBlock{
				message.ToolCall(forge.ToolCall{ID: "toolu_1", Name: "search", Arguments: json.RawMessage(`{"query":"go"}`)}),
			}},
			message.ToolMessage(forge.ToolResult{CallID: "toolu_1", Content: "two results"}),
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(got.Messages) != 3 {
		t.Fatalf("messages sent = %d, want 3", len(got.Messages))
	}
	assistant := got.Messages[1]
	if assistant.Role != "assistant" || len(assistant.Content) != 1 || assistant.Content[0].Type != "tool_use" {
		t.Errorf("assistant message = %+v, want a single tool_use block", assistant)
	}
	result := got.Messages[2]
	if result.Role != "user" {
		t.Errorf("tool result role = %q, want user", result.Role)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "tool_result" {
		t.Fatalf("tool result content = %+v, want one tool_result block", result.Content)
	}
	if result.Content[0].ToolUseID != "toolu_1" {
		t.Errorf("tool_use_id = %q, want toolu_1", result.Content[0].ToolUseID)
	}
}

func TestMaxTokensDefaultAndOverride(t *testing.T) {
	if p := New("k", ModelClaudeSonnet5); p.maxTokens != defaultMaxTokens {
		t.Errorf("default maxTokens = %d, want %d", p.maxTokens, defaultMaxTokens)
	}
	if p := New("k", ModelClaudeSonnet5, WithMaxTokens(64000)); p.maxTokens != 64000 {
		t.Errorf("maxTokens = %d, want 64000", p.maxTokens)
	}
	// A non-positive value would make every request fail; it must be ignored.
	if p := New("k", ModelClaudeSonnet5, WithMaxTokens(0)); p.maxTokens != defaultMaxTokens {
		t.Errorf("maxTokens = %d, want default retained for zero", p.maxTokens)
	}
}

func TestMaxTokensReachesTheWire(t *testing.T) {
	var got toolRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "m", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"stop_reason": "end_turn",
			"content":     []map[string]any{{"type": "text", "text": "hi"}},
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelClaudeSonnet5, WithBaseURL(srv.URL), WithMaxTokens(2048))
	if _, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{message.UserText("hi")},
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.MaxTokens != 2048 {
		t.Errorf("max_tokens = %d, want 2048", got.MaxTokens)
	}
}

type toolRequest struct {
	MaxTokens int               `json:"max_tokens"`
	Tools     []wireTool        `json:"tools"`
	Messages  []wireToolMessage `json:"messages"`
}

type wireTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	} `json:"input_schema"`
}

type wireToolMessage struct {
	Role    string          `json:"role"`
	Content []wireToolBlock `json:"content"`
}

type wireToolBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	ToolUseID string `json:"tool_use_id"`
	ID        string `json:"id"`
	Name      string `json:"name"`
}
