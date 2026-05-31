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

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "claude-sonnet-4-20250514" {
			t.Errorf("model = %q, want %q", req.Model, "claude-sonnet-4-20250514")
		}
		if req.System != "You are helpful." {
			t.Errorf("system = %q, want %q", req.System, "You are helpful.")
		}

		resp := response{
			Content:    []content{{Type: "text", Text: "Hello!"}},
			StopReason: "end_turn",
			Usage:      usage{InputTokens: 10, OutputTokens: 5},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := New("test-key", "claude-sonnet-4-20250514")
	// Override the endpoint to use the test server.
	p.client = srv.Client()

	// We need to redirect requests to our test server. Create a custom transport.
	p.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = srv.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(r)
	})

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

	p := New("bad-key", "claude-sonnet-4-20250514")
	p.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			r.URL.Scheme = "http"
			r.URL.Host = srv.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(r)
		}),
	}

	_, err := p.Generate(context.Background(), forge.ProviderRequest{
		Messages: []forge.Message{forge.UserText("Hi")},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
