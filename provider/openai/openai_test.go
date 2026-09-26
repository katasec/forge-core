package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/katasec/forge-core"
	"github.com/katasec/forge-core/message"
)

// Compile-time check that *OpenAIProvider satisfies forge.Provider.
var _ forge.Provider = (*OpenAIProvider)(nil)

func TestNew(t *testing.T) {
	p := New("test-key", ModelGPT54Nano)
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.baseURL != "https://api.openai.com/v1" {
		t.Errorf("baseURL = %q, want trimmed base URL", p.baseURL)
	}
	if p.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want test-key", p.apiKey)
	}
	if p.model != "gpt-5.4-nano" {
		t.Errorf("model = %q, want gpt-5.4-nano", p.model)
	}
}

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/responses" {
			t.Errorf("path = %q, want /responses", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt-5.4-nano" {
			t.Errorf("model = %q, want gpt-5.4-nano", req.Model)
		}
		if req.Instructions != "You are helpful." {
			t.Errorf("instructions = %q", req.Instructions)
		}
		if len(req.Input) != 1 || req.Input[0].Role != "user" {
			t.Fatalf("input = %+v, want one user item", req.Input)
		}
		if got := req.Input[0].Content[0]; got.Type != "input_text" || got.Text != "Hi" {
			t.Fatalf("content = %+v, want input_text Hi", got)
		}

		resp := response{
			Output: []outputItem{{
				Type: "message",
				Role: "assistant",
				Content: []contentOutput{{
					Type: "output_text",
					Text: "Hello!",
				}},
			}},
			Usage: usage{
				InputTokens:  8,
				OutputTokens: 3,
				TotalTokens:  11,
				InputTokensDetails: inputTokensDetails{
					CachedTokens: 2,
				},
				OutputTokensDetails: outputTokensDetails{
					ReasoningTokens: 1,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))

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
		t.Errorf("content = %q, want Hello!", resp.Messages[0].Text())
	}
	if resp.Messages[0].Role != forge.RoleAssistant {
		t.Errorf("role = %q, want %q", resp.Messages[0].Role, forge.RoleAssistant)
	}
	if resp.FinishReason != forge.FinishReasonStop {
		t.Errorf("finishReason = %q, want %q", resp.FinishReason, forge.FinishReasonStop)
	}
	if resp.Usage.InputTokens != 8 || resp.Usage.OutputTokens != 3 || resp.Usage.TotalTokens != 11 {
		t.Errorf("usage = %+v, want input 8 output 3 total 11", resp.Usage)
	}
	if resp.Usage.CachedInputTokens != 2 || resp.Usage.ReasoningOutputTokens != 1 {
		t.Errorf("usage details = %+v, want cached 2 reasoning 1", resp.Usage)
	}
}

func TestGenerateWithImageURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Input) != 1 || len(req.Input[0].Content) != 2 {
			t.Fatalf("input content = %+v, want text and image", req.Input)
		}
		image := req.Input[0].Content[1]
		if image.Type != "input_image" || image.ImageURL != "https://example.com/cat.png" {
			t.Fatalf("image content = %+v", image)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{
			Output: []outputItem{{
				Type:    "message",
				Role:    "assistant",
				Content: []contentOutput{{Type: "output_text", Text: "A cat."}},
			}},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))
	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{
			message.UserMessage(
				message.Text("Describe this image."),
				message.ImageURL("https://example.com/cat.png"),
			),
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Messages[0].Text() != "A cat." {
		t.Errorf("content = %q, want A cat.", resp.Messages[0].Text())
	}
}

func TestGenerateNoMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := response{Output: []outputItem{}, Usage: usage{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))

	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{message.UserText("Hi")},
	})
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))

	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{message.UserText("Hi")},
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}

type request struct {
	Model        string      `json:"model"`
	Input        []inputItem `json:"input"`
	Instructions string      `json:"instructions"`
}

type inputItem struct {
	Role    string         `json:"role"`
	Content []contentInput `json:"content"`
}

type contentInput struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type response struct {
	Output []outputItem `json:"output"`
	Usage  usage        `json:"usage"`
}

type outputItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content []contentOutput `json:"content"`
}

type contentOutput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens         int                 `json:"input_tokens"`
	OutputTokens        int                 `json:"output_tokens"`
	TotalTokens         int                 `json:"total_tokens"`
	InputTokensDetails  inputTokensDetails  `json:"input_tokens_details"`
	OutputTokensDetails outputTokensDetails `json:"output_tokens_details"`
}

type inputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type outputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// TestGenerateSendsToolsAndParsesFunctionCall covers the OpenAI tool round
// trip: definitions reach the wire and a function_call output becomes a
// Forge ToolCall.
func TestGenerateSendsToolsAndParsesFunctionCall(t *testing.T) {
	var got toolWireRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{{
				"type": "function_call", "call_id": "call_1",
				"name": "search", "arguments": `{"query":"go"}`,
			}},
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))
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
	if got.Tools[0].Name != "search" || got.Tools[0].Type != "function" {
		t.Errorf("tool = %+v, want a function tool named search", got.Tools[0])
	}
	if _, ok := got.Tools[0].Parameters["properties"]; !ok {
		t.Errorf("parameters = %v, want a schema with properties", got.Tools[0].Parameters)
	}

	if resp.FinishReason != forge.FinishReasonToolUse {
		t.Errorf("finish reason = %q, want %q", resp.FinishReason, forge.FinishReasonToolUse)
	}
	calls := resp.Messages[0].ToolCalls()
	if len(calls) != 1 || calls[0].ID != "call_1" || calls[0].Name != "search" {
		t.Fatalf("calls = %+v, want one call_1/search", calls)
	}
	if string(calls[0].Arguments) != `{"query":"go"}` {
		t.Errorf("arguments = %s, want {\"query\":\"go\"}", calls[0].Arguments)
	}
}

// TestGenerateSendsToolResults checks a tool result is replayed as its own
// function_call_output input item carrying the originating call id.
func TestGenerateSendsToolResults(t *testing.T) {
	var got toolWireRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{{
				"type": "message", "role": "assistant",
				"content": []map[string]any{{"type": "output_text", "text": "done"}},
			}},
			"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))
	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{
			message.UserText("find something"),
			{Role: forge.RoleAssistant, Content: []forge.ContentBlock{
				message.ToolCall(forge.ToolCall{ID: "call_1", Name: "search", Arguments: json.RawMessage(`{"query":"go"}`)}),
			}},
			message.ToolMessage(forge.ToolResult{CallID: "call_1", Content: "two results"}),
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(got.Input) != 3 {
		t.Fatalf("input items = %d, want 3", len(got.Input))
	}
	if got.Input[1].Type != "function_call" || got.Input[1].CallID != "call_1" {
		t.Errorf("item[1] = %+v, want function_call for call_1", got.Input[1])
	}
	if got.Input[2].Type != "function_call_output" || got.Input[2].CallID != "call_1" {
		t.Errorf("item[2] = %+v, want function_call_output for call_1", got.Input[2])
	}
	if got.Input[2].Output != "two results" {
		t.Errorf("output = %q, want %q", got.Input[2].Output, "two results")
	}
}

type toolWireRequest struct {
	Tools []toolWire      `json:"tools"`
	Input []toolWireInput `json:"input"`
}

type toolWire struct {
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

type toolWireInput struct {
	Type   string `json:"type"`
	Role   string `json:"role"`
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	Output string `json:"output"`
}

// TestGenerateReplaysAssistantHistory guards multi-turn conversations: OpenAI
// rejects input_text on an assistant message ("Supported values are:
// 'output_text' and 'refusal'"), so a remembered assistant turn must not be
// sent as input_text content parts.
func TestGenerateReplaysAssistantHistory(t *testing.T) {
	var got struct {
		Input []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"input"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{{
				"type": "message", "role": "assistant",
				"content": []map[string]any{{"type": "output_text", "text": "sure"}},
			}},
			"usage": map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer srv.Close()

	p := New("test-key", ModelGPT54Nano, WithBaseURL(srv.URL))
	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{
			message.UserText("who made you?"),
			message.AssistantText("OpenAI made me."),
			message.UserText("what is 21 + 21?"),
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(got.Input) != 3 {
		t.Fatalf("input items = %d, want 3", len(got.Input))
	}
	assistant := got.Input[1]
	if assistant.Role != "assistant" {
		t.Fatalf("item[1] role = %q, want assistant", assistant.Role)
	}
	if strings.Contains(string(assistant.Content), "input_text") {
		t.Errorf("assistant content = %s, must not use input_text", assistant.Content)
	}
	// The plain-string form lets the API choose the right content type.
	var text string
	if err := json.Unmarshal(assistant.Content, &text); err != nil {
		t.Errorf("assistant content = %s, want a plain JSON string: %v", assistant.Content, err)
	}
	if text != "OpenAI made me." {
		t.Errorf("assistant text = %q, want %q", text, "OpenAI made me.")
	}
	// A user turn still uses input_text content parts.
	if !strings.Contains(string(got.Input[0].Content), "input_text") {
		t.Errorf("user content = %s, want input_text parts", got.Input[0].Content)
	}
}
