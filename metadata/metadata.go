package metadata

import "context"

type key struct{}

// Metadata holds arbitrary key-value pairs attached to a context.
type Metadata struct {
	Values map[string]string
}

// WithMetadata returns a new context with the given Metadata stored in it.
func WithMetadata(ctx context.Context, m Metadata) context.Context {
	if m.Values == nil {
		m.Values = make(map[string]string)
	}
	return context.WithValue(ctx, key{}, m)
}

// FromContext retrieves Metadata from the context.
// Returns false if no Metadata is present.
func FromContext(ctx context.Context) (Metadata, bool) {
	m, ok := ctx.Value(key{}).(Metadata)
	return m, ok
}
