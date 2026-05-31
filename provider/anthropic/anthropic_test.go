package anthropic

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
	p := New("test-key", "claude-sonnet-4-20250514")
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", p.apiKey, "test-key")
	}
	if p.model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", p.model, "claude-sonnet-4-20250514")
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
		if req.Model != "claude-sonnet-4-20250514" {
			t.Errorf("model = %q, want %q", req.Model, "claude-sonnet-4-20250514")
		}
		if len(req.System) != 1 || req.System[0].Text != "You are helpful." {
			t.Errorf("system = %q, want %q", req.System, "You are helpful.")
		}

		resp := apiResponse{
			ID:         "msg_test",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-sonnet-4-20250514",
			Content:    []contentBlock{{Type: "text", Text: "Hello!"}},
			StopReason: "end_turn",
			Usage:      usageBlock{InputTokens: 10, OutputTokens: 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("test-key", "claude-sonnet-4-20250514", WithBaseURL(srv.URL))

	resp, err := p.Generate(context.Background(), forge.ProviderRequest{
		SystemPrompt: "You are helpful.",
		Messages: []forge.Message{
			forge.UserText("Hi"),
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

	p := New("bad-key", "claude-sonnet-4-20250514", WithBaseURL(srv.URL))

	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{forge.UserText("Hi")},
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
