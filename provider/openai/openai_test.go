package openai

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
			forge.UserText("Hi"),
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
			forge.UserMessage(
				forge.Text("Describe this image."),
				forge.ImageURL("https://example.com/cat.png"),
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
		Messages: []forge.Message{forge.UserText("Hi")},
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
		Messages: []forge.Message{forge.UserText("Hi")},
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}
