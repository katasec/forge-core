package forge

import (
	"context"

	toolpkg "github.com/katasec/forge-core/tool"
)

type Tool = toolpkg.Tool
type ToolSchema = toolpkg.Schema
type ToolDefinition = toolpkg.Definition

func Func[T any](name, description string, fn func(ctx context.Context, input T) (string, error)) Tool {
	return toolpkg.Func(name, description, fn)
}
