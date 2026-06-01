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
type funcTool[In any, Out any] struct {
	name        string
	description string
	schema      Schema
	fn          func(ctx context.Context, input In) (Out, error)
}

// Func creates a Tool from a typed function. The JSON Schema for parameters
// is derived from In using invopop/jsonschema at construction time.
func Func[In any, Out any](name, description string, fn func(ctx context.Context, input In) (Out, error)) Tool {
	r := new(jsonschema.Reflector)
	s := r.Reflect(new(In))
	params, _ := json.Marshal(s)

	return &funcTool[In, Out]{
		name:        name,
		description: description,
		schema:      Schema{Parameters: params},
		fn:          fn,
	}
}

func (t *funcTool[In, Out]) Name() string        { return t.name }
func (t *funcTool[In, Out]) Description() string { return t.description }
func (t *funcTool[In, Out]) Schema() Schema      { return t.schema }

func (t *funcTool[In, Out]) Invoke(ctx context.Context, args json.RawMessage) (string, error) {
	var input In
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	output, err := t.fn(ctx, input)
	if err != nil {
		return "", err
	}
	return encodeOutput(output)
}

func encodeOutput(output any) (string, error) {
	switch v := output.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		data, err := json.Marshal(output)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}
