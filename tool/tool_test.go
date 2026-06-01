package tool

import (
	"context"
	"encoding/json"
	"testing"
)

type addInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

func TestFuncSchemaGeneration(t *testing.T) {
	tool := Func[addInput, string]("add", "adds two numbers", func(_ context.Context, in addInput) (string, error) {
		return "", nil
	})

	if tool.Name() != "add" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "add")
	}
	if tool.Description() != "adds two numbers" {
		t.Errorf("Description() = %q, want %q", tool.Description(), "adds two numbers")
	}

	schema := tool.Schema()
	if len(schema.Parameters) == 0 {
		t.Fatal("expected non-empty schema parameters")
	}

	props := schemaProperties(t, schema.Parameters)
	if _, ok := props["a"]; !ok {
		t.Error("expected property 'a' in schema")
	}
	if _, ok := props["b"]; !ok {
		t.Error("expected property 'b' in schema")
	}
}

func TestFuncInvoke(t *testing.T) {
	tool := Func[addInput, int]("add", "adds two numbers", func(_ context.Context, in addInput) (int, error) {
		return in.A + in.B, nil
	})

	args := json.RawMessage(`{"a": 2, "b": 3}`)
	result, err := tool.Invoke(context.Background(), args)
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result != "5" {
		t.Errorf("Invoke result = %q, want %q", result, "5")
	}
}

func TestFuncInvokeStringOutput(t *testing.T) {
	tool := Func[addInput, string]("add", "adds two numbers", func(_ context.Context, in addInput) (string, error) {
		return "sum", nil
	})

	result, err := tool.Invoke(context.Background(), json.RawMessage(`{"a": 2, "b": 3}`))
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result != "sum" {
		t.Errorf("Invoke result = %q, want %q", result, "sum")
	}
}

func TestFuncInvokeBytesOutput(t *testing.T) {
	tool := Func[addInput, []byte]("add", "adds two numbers", func(_ context.Context, in addInput) ([]byte, error) {
		return []byte("bytes"), nil
	})

	result, err := tool.Invoke(context.Background(), json.RawMessage(`{"a": 2, "b": 3}`))
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result != "bytes" {
		t.Errorf("Invoke result = %q, want %q", result, "bytes")
	}
}

func TestFuncInvokeStructOutput(t *testing.T) {
	type addOutput struct {
		Sum int `json:"sum"`
	}

	tool := Func[addInput, addOutput]("add", "adds two numbers", func(_ context.Context, in addInput) (addOutput, error) {
		return addOutput{Sum: in.A + in.B}, nil
	})

	result, err := tool.Invoke(context.Background(), json.RawMessage(`{"a": 2, "b": 3}`))
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result != `{"sum":5}` {
		t.Errorf("Invoke result = %q, want %q", result, `{"sum":5}`)
	}
}

func TestFuncInvokeInvalidArgs(t *testing.T) {
	tool := Func[addInput, string]("add", "adds two numbers", func(_ context.Context, in addInput) (string, error) {
		return "", nil
	})

	args := json.RawMessage(`not json`)
	_, err := tool.Invoke(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for invalid JSON args")
	}
}

func schemaProperties(t *testing.T, parameters json.RawMessage) map[string]interface{} {
	t.Helper()

	var s map[string]interface{}
	if err := json.Unmarshal(parameters, &s); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	defs, ok := s["$defs"].(map[string]interface{})
	if !ok {
		t.Fatal("expected $defs in schema")
	}
	def, ok := defs["addInput"].(map[string]interface{})
	if !ok {
		t.Fatal("expected addInput in $defs")
	}
	props, ok := def["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties in addInput definition")
	}
	return props
}
