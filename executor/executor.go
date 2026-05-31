package executor

import (
	"context"

	"github.com/katasec/forge-core/tool"
)

// Executor executes a batch of tool calls and returns the results.
type Executor interface {
	Execute(ctx context.Context, calls []tool.Call) []tool.Result
}
