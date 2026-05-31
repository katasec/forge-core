package tool

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// Tool defines a callable tool that can be invoked by an agent.
type Tool interface {
	Name() string
	Description() string
	Schema() Schema
	Invoke(ctx context.Context, args json.RawMessage) (string, error)
}

// Schema describes the JSON Schema for a tool's parameters.
type Schema struct {
	Parameters json.RawMessage `json:"parameters"`
}

// Definition is the wire format sent to providers so they know which tools are available.
type Definition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      Schema `json:"schema"`
}

// funcTool is the unexported implementation backing the Func helper.
type funcTool[T any] struct {
	name        string
	description string
	schema      Schema
	fn          func(ctx context.Context, input T) (string, error)
}

// Func creates a Tool from a typed function. The JSON Schema for parameters
// is derived from T using invopop/jsonschema at construction time.
func Func[T any](name, description string, fn func(ctx context.Context, input T) (string, error)) Tool {
	r := new(jsonschema.Reflector)
	s := r.Reflect(new(T))
	params, _ := json.Marshal(s)

	return &funcTool[T]{
		name:        name,
		description: description,
		schema:      Schema{Parameters: params},
		fn:          fn,
	}
}

func (t *funcTool[T]) Name() string        { return t.name }
func (t *funcTool[T]) Description() string { return t.description }
func (t *funcTool[T]) Schema() Schema      { return t.schema }

func (t *funcTool[T]) Invoke(ctx context.Context, args json.RawMessage) (string, error) {
	var input T
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	return t.fn(ctx, input)
}
