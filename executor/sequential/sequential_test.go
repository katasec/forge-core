package sequential

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/katasec/forge-core/tool"
	"github.com/katasec/forge-core/tool/registry"
)

type addInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

func TestExecutorSuccess(t *testing.T) {
	r := registry.New()
	r.Register(addTool())

	exec := &Executor{Registry: r}
	results := exec.Execute(context.Background(), []tool.Call{
		{ID: "call-1", Name: "add", Arguments: json.RawMessage(`{"a": 1, "b": 2}`)},
	})

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].IsError {
		t.Fatalf("expected success, got error: %s", results[0].Content)
	}
	if results[0].Content != "3" {
		t.Errorf("Content = %q, want %q", results[0].Content, "3")
	}
	if results[0].CallID != "call-1" {
		t.Errorf("CallID = %q, want %q", results[0].CallID, "call-1")
	}
}

func TestExecutorMissingTool(t *testing.T) {
	exec := &Executor{Registry: registry.New()}

	results := exec.Execute(context.Background(), []tool.Call{
		{ID: "call-1", Name: "nonexistent", Arguments: json.RawMessage(`{}`)},
	})

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].IsError {
		t.Fatal("expected error for missing tool")
	}
	if results[0].Content != "tool not found: nonexistent" {
		t.Errorf("Content = %q", results[0].Content)
	}
}

func TestExecutorToolError(t *testing.T) {
	r := registry.New()
	r.Register(failingTool())

	exec := &Executor{Registry: r}
	results := exec.Execute(context.Background(), []tool.Call{
		{ID: "call-1", Name: "fail", Arguments: json.RawMessage(`{"a":0,"b":0}`)},
	})

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].IsError {
		t.Fatal("expected error result")
	}
	if results[0].Content != "something broke" {
		t.Errorf("Content = %q, want %q", results[0].Content, "something broke")
	}
}

func TestExecutorMultipleCalls(t *testing.T) {
	r := registry.New()
	r.Register(addTool())

	exec := &Executor{Registry: r}
	results := exec.Execute(context.Background(), []tool.Call{
		{ID: "c1", Name: "add", Arguments: json.RawMessage(`{"a": 1, "b": 2}`)},
		{ID: "c2", Name: "add", Arguments: json.RawMessage(`{"a": 10, "b": 20}`)},
	})

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Content != "3" {
		t.Errorf("results[0].Content = %q, want %q", results[0].Content, "3")
	}
	if results[1].Content != "30" {
		t.Errorf("results[1].Content = %q, want %q", results[1].Content, "30")
	}
}

func addTool() tool.Tool {
	return tool.Func[addInput, string]("add", "adds", func(_ context.Context, in addInput) (string, error) {
		b, err := json.Marshal(in.A + in.B)
		if err != nil {
			return "", err
		}
		return string(b), nil
	})
}

func failingTool() tool.Tool {
	return tool.Func[addInput, string]("fail", "fails", func(_ context.Context, _ addInput) (string, error) {
		return "", errors.New("something broke")
	})
}
