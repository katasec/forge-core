package sequential

import (
	"context"
	"fmt"

	"github.com/katasec/forge-core/tool"
	"github.com/katasec/forge-core/tool/registry"
)

// Executor invokes tools one at a time via a tool registry.
type Executor struct {
	Registry *registry.Registry
}

// Execute processes each tool call in order. Missing tools and invocation
// errors are returned as Results with IsError set to true.
func (e *Executor) Execute(ctx context.Context, calls []tool.Call) []tool.Result {
	results := make([]tool.Result, 0, len(calls))
	for _, call := range calls {
		t, ok := e.Registry.Get(call.Name)
		if !ok {
			results = append(results, tool.Result{
				CallID:  call.ID,
				Content: fmt.Sprintf("tool not found: %s", call.Name),
				IsError: true,
			})
			continue
		}

		content, err := t.Invoke(ctx, call.Arguments)
		if err != nil {
			results = append(results, tool.Result{
				CallID:  call.ID,
				Content: err.Error(),
				IsError: true,
			})
			continue
		}

		results = append(results, tool.Result{
			CallID:  call.ID,
			Content: content,
			IsError: false,
		})
	}
	return results
}
