package forge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestSequentialExecutorSuccess(t *testing.T) {
	r := NewToolRegistry()
	r.Register(Func[addInput]("add", "adds", func(_ context.Context, in addInput) (string, error) {
		b, _ := json.Marshal(in.A + in.B)
		return string(b), nil
	}))

	exec := &SequentialExecutor{Registry: r}
	results := exec.Execute(context.Background(), []ToolCall{
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

func TestSequentialExecutorMissingTool(t *testing.T) {
	r := NewToolRegistry()
	exec := &SequentialExecutor{Registry: r}

	results := exec.Execute(context.Background(), []ToolCall{
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

func TestSequentialExecutorToolError(t *testing.T) {
	r := NewToolRegistry()
	r.Register(Func[addInput]("fail", "fails", func(_ context.Context, _ addInput) (string, error) {
		return "", errors.New("something broke")
	}))

	exec := &SequentialExecutor{Registry: r}
	results := exec.Execute(context.Background(), []ToolCall{
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

func TestSequentialExecutorMultipleCalls(t *testing.T) {
	r := NewToolRegistry()
	r.Register(Func[addInput]("add", "adds", func(_ context.Context, in addInput) (string, error) {
		b, _ := json.Marshal(in.A + in.B)
		return string(b), nil
	}))

	exec := &SequentialExecutor{Registry: r}
	results := exec.Execute(context.Background(), []ToolCall{
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
