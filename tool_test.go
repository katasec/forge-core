package forge

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
	tool := Func[addInput]("add", "adds two numbers", func(_ context.Context, in addInput) (string, error) {
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

	// Verify the schema contains our fields.
	// invopop/jsonschema uses $ref + $defs, so we follow the reference.
	var s map[string]interface{}
	if err := json.Unmarshal(schema.Parameters, &s); err != nil {
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
	if _, ok := props["a"]; !ok {
		t.Error("expected property 'a' in schema")
	}
	if _, ok := props["b"]; !ok {
		t.Error("expected property 'b' in schema")
	}
}

func TestFuncInvoke(t *testing.T) {
	tool := Func[addInput]("add", "adds two numbers", func(_ context.Context, in addInput) (string, error) {
		sum := in.A + in.B
		b, _ := json.Marshal(sum)
		return string(b), nil
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

func TestFuncInvokeInvalidArgs(t *testing.T) {
	tool := Func[addInput]("add", "adds two numbers", func(_ context.Context, in addInput) (string, error) {
		return "", nil
	})

	args := json.RawMessage(`not json`)
	_, err := tool.Invoke(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for invalid JSON args")
	}
}

func TestUserMessage(t *testing.T) {
	msg := UserText("hello")
	if msg.Role != RoleUser {
		t.Fatalf("Role = %q, want %q", msg.Role, RoleUser)
	}
	if msg.Text() != "hello" {
		t.Fatalf("Content = %q, want hello", msg.Text())
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewToolRegistry()
	tool := Func[addInput]("add", "adds", func(_ context.Context, in addInput) (string, error) {
		return "", nil
	})

	r.Register(tool)

	got, ok := r.Get("add")
	if !ok {
		t.Fatal("expected to find tool 'add'")
	}
	if got.Name() != "add" {
		t.Errorf("Name() = %q, want %q", got.Name(), "add")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewToolRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("expected tool not found")
	}
}

func TestRegistryDefinitions(t *testing.T) {
	r := NewToolRegistry()
	r.Register(
		Func[addInput]("add", "adds", func(_ context.Context, _ addInput) (string, error) { return "", nil }),
		Func[addInput]("sub", "subtracts", func(_ context.Context, _ addInput) (string, error) { return "", nil }),
	)

	defs := r.Definitions()
	if len(defs) != 2 {
		t.Fatalf("got %d definitions, want 2", len(defs))
	}
	if defs[0].Name != "add" {
		t.Errorf("defs[0].Name = %q, want %q", defs[0].Name, "add")
	}
	if defs[1].Name != "sub" {
		t.Errorf("defs[1].Name = %q, want %q", defs[1].Name, "sub")
	}
}

func TestRegistryDuplicateOverwrite(t *testing.T) {
	r := NewToolRegistry()
	r.Register(Func[addInput]("add", "v1", func(_ context.Context, _ addInput) (string, error) { return "", nil }))
	r.Register(Func[addInput]("add", "v2", func(_ context.Context, _ addInput) (string, error) { return "", nil }))

	got, ok := r.Get("add")
	if !ok {
		t.Fatal("expected to find tool 'add'")
	}
	if got.Description() != "v2" {
		t.Errorf("Description() = %q, want %q (last-write-wins)", got.Description(), "v2")
	}

	// Should still be only one definition.
	defs := r.Definitions()
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
}
