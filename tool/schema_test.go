package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestObjectSchemaResolvesRef(t *testing.T) {
	// The shape invopop/jsonschema emits for a reflected struct.
	raw := json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$ref": "#/$defs/SearchInput",
		"$defs": {
			"SearchInput": {
				"type": "object",
				"properties": {"query": {"type": "string"}},
				"required": ["query"]
			}
		}
	}`)

	schema := ObjectSchema(raw)
	if schema["type"] != "object" {
		t.Fatalf("type = %v, want object", schema["type"])
	}
	if _, ok := SchemaProperties(schema)["query"]; !ok {
		t.Error("expected resolved schema to expose the 'query' property")
	}
	if got := SchemaRequired(schema); len(got) != 1 || got[0] != "query" {
		t.Errorf("required = %v, want [query]", got)
	}
}

func TestObjectSchemaPassesThroughPlainObject(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","properties":{"a":{"type":"integer"}}}`)

	if _, ok := SchemaProperties(ObjectSchema(raw))["a"]; !ok {
		t.Error("expected plain object schema to be returned unchanged")
	}
}

func TestObjectSchemaToleratesBadInput(t *testing.T) {
	if schema := ObjectSchema(json.RawMessage(`not json`)); schema != nil {
		t.Errorf("schema = %v, want nil for unparseable input", schema)
	}
	if got := SchemaRequired(map[string]any{"required": "not-a-list"}); len(got) != 0 {
		t.Errorf("required = %v, want empty for malformed input", got)
	}
}

// TestFuncSchemaResolves guards the real pipeline: a schema built by Func must
// survive ObjectSchema, since that is what providers send on the wire.
func TestFuncSchemaResolves(t *testing.T) {
	type in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	tl := Func[in, string]("search", "search", func(_ context.Context, _ in) (string, error) {
		return "", nil
	})

	props := SchemaProperties(ObjectSchema(tl.Schema().Parameters))
	for _, want := range []string{"query", "limit"} {
		if _, ok := props[want]; !ok {
			t.Errorf("expected property %q in resolved schema, got %v", want, props)
		}
	}
}
