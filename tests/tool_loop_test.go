package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/katasec/kiln"
	"github.com/katasec/kiln/provider/anthropic"
	"github.com/katasec/kiln/tool"
)

type addInput struct {
	A int `json:"a" jsonschema:"description=first addend"`
	B int `json:"b" jsonschema:"description=second addend"`
}

// TestAnthropicToolLoopEndToEnd drives the whole contract: an agent with a
// registered tool, an Anthropic provider, and a fake API that asks for the tool
// on turn one and answers on turn two. It fails if the provider drops tools,
// mangles the schema, or cannot replay tool results.
func TestAnthropicToolLoopEndToEnd(t *testing.T) {
	var turns int
	var sawSchema bool
	var sawToolResult string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]any `json:"properties"`
				} `json:"input_schema"`
			} `json:"tools"`
			Messages []struct {
				Role    string `json:"role"`
				Content []struct {
					Type    string          `json:"type"`
					Content json.RawMessage `json:"content"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		turns++

		// The model can only call the tool if the schema survived the trip.
		if len(req.Tools) == 1 && req.Tools[0].Name == "add" {
			if _, ok := req.Tools[0].InputSchema.Properties["a"]; ok {
				sawSchema = true
			}
		}
		for _, m := range req.Messages {
			for _, c := range m.Content {
				if c.Type == "tool_result" {
					sawToolResult = string(c.Content)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if turns == 1 {
			json.NewEncoder(w).Encode(map[string]any{
				"id": "m1", "type": "message", "role": "assistant",
				"model": "claude-sonnet-5", "stop_reason": "tool_use",
				"content": []map[string]any{{
					"type": "tool_use", "id": "toolu_1", "name": "add",
					"input": map[string]any{"a": 2, "b": 3},
				}},
				"usage": map[string]any{"input_tokens": 10, "output_tokens": 5},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id": "m2", "type": "message", "role": "assistant",
			"model": "claude-sonnet-5", "stop_reason": "end_turn",
			"content": []map[string]any{{"type": "text", "text": "The answer is 5."}},
			"usage":   map[string]any{"input_tokens": 12, "output_tokens": 6},
		})
	}))
	defer srv.Close()

	var invoked bool
	add := tool.Func[addInput, int]("add", "adds two numbers",
		func(_ context.Context, in addInput) (int, error) {
			invoked = true
			return in.A + in.B, nil
		})

	agent, err := kiln.NewAgent(kiln.Config{
		Provider:      anthropic.New("test-key", anthropic.ModelClaudeSonnet5, anthropic.WithBaseURL(srv.URL)),
		Tools:         []kiln.Tool{add},
		MaxIterations: 5,
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}

	resp, err := agent.Ask(context.Background(), "what is 2 + 3?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	if !sawSchema {
		t.Error("provider did not send the tool schema; the model could never call it")
	}
	if !invoked {
		t.Error("tool was never invoked")
	}
	if !strings.Contains(sawToolResult, "5") {
		t.Errorf("tool result sent back = %s, want it to carry the value 5", sawToolResult)
	}
	if turns != 2 {
		t.Errorf("provider calls = %d, want 2", turns)
	}
	if got := resp.LastText(); !strings.Contains(got, "5") {
		t.Errorf("final answer = %q, want it to mention 5", got)
	}
}
