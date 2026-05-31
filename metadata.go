package forge

import (
	"context"

	metadatapkg "github.com/katasec/forge-core/metadata"
)

type Metadata = metadatapkg.Metadata

func WithMetadata(ctx context.Context, m Metadata) context.Context {
	return metadatapkg.WithMetadata(ctx, m)
}

func MetadataFromContext(ctx context.Context) (Metadata, bool) {
	return metadatapkg.FromContext(ctx)
}
