package middleware

import (
	"context"

	"github.com/katasec/forge-core/provider"
)

// RunFunc is the signature for a single provider call, used by middleware.
type RunFunc func(ctx context.Context, req provider.Request) (*provider.Response, error)

// Middleware wraps a RunFunc to intercept provider calls.
// Composition order: given [A, B, C], request flows A -> B -> C -> provider -> C -> B -> A.
type Middleware func(next RunFunc) RunFunc
