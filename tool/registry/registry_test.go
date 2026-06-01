package registry

import (
	"context"
	"testing"

	"github.com/katasec/forge-core/tool"
)

func TestRegisterAndGet(t *testing.T) {
	r := New()
	tool := addTool("add", "adds")

	r.Register(tool)

	got, ok := r.Get("add")
	if !ok {
		t.Fatal("expected to find tool 'add'")
	}
	if got.Name() != "add" {
		t.Errorf("Name() = %q, want %q", got.Name(), "add")
	}
}

func TestGetMissing(t *testing.T) {
	r := New()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("expected tool not found")
	}
}

func TestDefinitions(t *testing.T) {
	r := New()
	r.Register(
		addTool("add", "adds"),
		addTool("sub", "subtracts"),
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

func TestDuplicateOverwrite(t *testing.T) {
	r := New()
	r.Register(addTool("add", "v1"))
	r.Register(addTool("add", "v2"))

	got, ok := r.Get("add")
	if !ok {
		t.Fatal("expected to find tool 'add'")
	}
	if got.Description() != "v2" {
		t.Errorf("Description() = %q, want %q (last-write-wins)", got.Description(), "v2")
	}

	defs := r.Definitions()
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
}

type addInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

func addTool(name, description string) tool.Tool {
	return tool.Func[addInput, string](name, description, func(_ context.Context, in addInput) (string, error) {
		return "", nil
	})
}
