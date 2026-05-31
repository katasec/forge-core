package xai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/katasec/forge-core"
)

// Compile-time check that *Provider satisfies forge.Provider.
var _ forge.Provider = (*Provider)(nil)

func TestNew(t *testing.T) {
	p := New("test-key", ModelGrok3Mini)
	if p.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", p.apiKey, "test-key")
	}
	if p.model != string(ModelGrok3Mini) {
		t.Errorf("model = %q, want %q", p.model, ModelGrok3Mini)
	}
	if p.baseURL != "https://api.x.ai/v1" {
		t.Errorf("baseURL = %q, want default", p.baseURL)
	}
	if len(p.tools) != 0 {
		t.Errorf("tools = %d, want 0", len(p.tools))
	}
}

func TestNewWithOptions(t *testing.T) {
	p := New("key", Model("custom-model"),
		WithBaseURL("http://localhost"),
		WithWebSearch(AllowedDomains("wikipedia.org", "github.com")),
		WithXSearch(ExcludedHandles("@spam")),
	)

	if p.baseURL != "http://localhost" {
		t.Errorf("baseURL = %q", p.baseURL)
	}
	if len(p.tools) != 2 {
		t.Fatalf("tools = %d, want 2", len(p.tools))
	}
	if p.tools[0].Type != "web_search" {
		t.Errorf("tools[0].Type = %q", p.tools[0].Type)
	}
	if len(p.tools[0].AllowedDomains) != 2 {
		t.Errorf("allowed_domains = %v", p.tools[0].AllowedDomains)
	}
	if p.tools[1].Type != "x_search" {
		t.Errorf("tools[1].Type = %q", p.tools[1].Type)
	}
	if len(p.tools[1].ExcludedHandles) != 1 {
		t.Errorf("excluded_handles = %v", p.tools[1].ExcludedHandles)
	}
}

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Path != "/responses" {
			t.Errorf("path = %q, want /responses", r.URL.Path)
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != string(ModelGrok3Mini) {
			t.Errorf("model = %q", req.Model)
		}
		if req.Instructions != "Be helpful." {
			t.Errorf("instructions = %q, want Be helpful.", req.Instructions)
		}
		if len(req.Input) != 1 {
			t.Fatalf("input items = %d, want 1", len(req.Input))
		}
		if req.Input[0].Role != "user" {
			t.Errorf("input[0].role = %q, want user", req.Input[0].Role)
		}

		resp := response{
			ID: "resp-123",
			Output: []outputItem{{
				Type: "message",
				Role: "assistant",
				Content: []contentItem{{
					Type: "output_text",
					Text: "Hello from Grok!",
				}},
			}},
			Usage: responseUsage{InputTokens: 10, OutputTokens: 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("test-key", ModelGrok3Mini, WithBaseURL(srv.URL))
	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		SystemPrompt: "Be helpful.",
		Messages: []forge.Message{
			forge.UserText("Hi"),
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.Messages[0].Text() != "Hello from Grok!" {
		t.Errorf("content = %q", resp.Messages[0].Text())
	}
	if resp.Messages[0].Role != forge.RoleAssistant {
		t.Errorf("role = %q", resp.Messages[0].Role)
	}
	if resp.FinishReason != forge.FinishReasonStop {
		t.Errorf("finishReason = %q", resp.FinishReason)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 5 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestGenerateWithFunctionCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}

		// Verify function tools are flat (not nested in "function" wrapper).
		if len(req.Tools) != 1 {
			t.Fatalf("tools = %d, want 1", len(req.Tools))
		}
		tool := req.Tools[0]
		if tool.Type != "function" {
			t.Errorf("tool.type = %q, want function", tool.Type)
		}
		if tool.Name != "get_weather" {
			t.Errorf("tool.name = %q", tool.Name)
		}
		if tool.Description != "Get weather" {
			t.Errorf("tool.description = %q", tool.Description)
		}

		resp := response{
			ID: "resp-456",
			Output: []outputItem{{
				Type:      "function_call",
				Name:      "get_weather",
				Arguments: `{"city":"SF"}`,
				CallID:    "call-1",
			}},
			Usage: responseUsage{InputTokens: 15, OutputTokens: 8},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("key", ModelGrok3Mini, WithBaseURL(srv.URL))
	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{forge.UserText("Weather in SF?")},
		Tools: []forge.ToolDefinition{{
			Name:        "get_weather",
			Description: "Get weather",
			Schema:      forge.ToolSchema{Parameters: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`)},
		}},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.FinishReason != forge.FinishReasonToolUse {
		t.Errorf("finishReason = %q, want tool_use", resp.FinishReason)
	}
	if len(resp.Messages[0].ToolCalls()) != 1 {
		t.Fatalf("toolCalls = %d, want 1", len(resp.Messages[0].ToolCalls()))
	}
	tc := resp.Messages[0].ToolCalls()[0]
	if tc.ID != "call-1" {
		t.Errorf("toolCall.ID = %q", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("toolCall.Name = %q", tc.Name)
	}
	if string(tc.Arguments) != `{"city":"SF"}` {
		t.Errorf("toolCall.Arguments = %s", tc.Arguments)
	}
}

func TestGenerateWithToolResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}

		// Find the function_call_output input item.
		var found bool
		for _, item := range req.Input {
			if item.Type == "function_call_output" {
				found = true
				if item.CallID != "call-1" {
					t.Errorf("call_id = %q", item.CallID)
				}
				if item.Output != "72°F" {
					t.Errorf("output = %q", item.Output)
				}
			}
		}
		if !found {
			t.Error("no function_call_output found in input")
		}

		resp := response{
			ID: "resp-789",
			Output: []outputItem{{
				Type: "message",
				Role: "assistant",
				Content: []contentItem{{
					Type: "output_text",
					Text: "It's 72°F in SF.",
				}},
			}},
			Usage: responseUsage{InputTokens: 20, OutputTokens: 10},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("key", ModelGrok3Mini, WithBaseURL(srv.URL))
	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{
			forge.UserText("Weather in SF?"),
			{Role: forge.RoleAssistant, Content: []forge.ContentBlock{
				forge.ToolCallBlock(forge.ToolCall{ID: "call-1", Name: "get_weather", Arguments: json.RawMessage(`{"city":"SF"}`)}),
			}},
			{Role: forge.RoleTool, Content: []forge.ContentBlock{
				forge.ToolResultBlock(forge.ToolResult{CallID: "call-1", Content: "72°F"}),
			}},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.Messages[0].Text() != "It's 72°F in SF." {
		t.Errorf("content = %q", resp.Messages[0].Text())
	}
	if resp.FinishReason != forge.FinishReasonStop {
		t.Errorf("finishReason = %q", resp.FinishReason)
	}
}

func TestGenerateWithWebSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}

		// Verify web_search tool is included.
		var hasWebSearch bool
		for _, tool := range req.Tools {
			if tool.Type == "web_search" {
				hasWebSearch = true
				if len(tool.AllowedDomains) != 1 || tool.AllowedDomains[0] != "reuters.com" {
					t.Errorf("allowed_domains = %v", tool.AllowedDomains)
				}
			}
		}
		if !hasWebSearch {
			t.Error("web_search tool not found in request")
		}

		resp := response{
			ID: "resp-search",
			Output: []outputItem{
				{Type: "web_search_call"}, // server-side, should be ignored
				{
					Type: "message",
					Role: "assistant",
					Content: []contentItem{{
						Type: "output_text",
						Text: "According to Reuters, xAI launched...",
						Annotations: []annotation{{
							Type:       "url_citation",
							URL:        "https://reuters.com/tech/xai",
							Title:      "1",
							StartIndex: 22,
							EndIndex:   35,
						}},
					}},
				},
			},
			Usage: responseUsage{InputTokens: 50, OutputTokens: 30},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("key", ModelGrok3Mini,
		WithBaseURL(srv.URL),
		WithWebSearch(AllowedDomains("reuters.com")),
	)
	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{forge.UserText("Latest xAI news?")},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if resp.Messages[0].Text() != "According to Reuters, xAI launched..." {
		t.Errorf("content = %q", resp.Messages[0].Text())
	}
	if resp.FinishReason != forge.FinishReasonStop {
		t.Errorf("finishReason = %q", resp.FinishReason)
	}

	// Check citations via provider-specific accessor.
	citations := p.LastCitations()
	if len(citations) != 1 {
		t.Fatalf("citations = %d, want 1", len(citations))
	}
	c := citations[0]
	if c.URL != "https://reuters.com/tech/xai" {
		t.Errorf("citation.URL = %q", c.URL)
	}
	if c.Source != "web" {
		t.Errorf("citation.Source = %q, want web", c.Source)
	}
	if c.StartIndex != 22 || c.EndIndex != 35 {
		t.Errorf("citation indices = [%d, %d]", c.StartIndex, c.EndIndex)
	}
}

func TestGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	p := New("key", ModelGrok3Mini, WithBaseURL(srv.URL))
	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{forge.UserText("Hi")},
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}

type request struct {
	Model        string        `json:"model"`
	Input        []inputItem   `json:"input"`
	Instructions string        `json:"instructions"`
	Tools        []requestTool `json:"tools,omitempty"`
}

type inputItem struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Type    string `json:"type,omitempty"`
	CallID  string `json:"call_id,omitempty"`
	Output  string `json:"output,omitempty"`
}
