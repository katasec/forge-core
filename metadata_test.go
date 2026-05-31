package forge

import (
	"context"
	"testing"
)

func TestWithMetadataAndFromContext(t *testing.T) {
	ctx := context.Background()
	m := Metadata{Values: map[string]string{"key": "value"}}

	ctx = WithMetadata(ctx, m)

	got, ok := MetadataFromContext(ctx)
	if !ok {
		t.Fatal("expected metadata in context")
	}
	if got.Values["key"] != "value" {
		t.Errorf("got %q, want %q", got.Values["key"], "value")
	}
}

func TestMetadataFromContextMissing(t *testing.T) {
	ctx := context.Background()

	_, ok := MetadataFromContext(ctx)
	if ok {
		t.Fatal("expected no metadata in context")
	}
}

func TestWithMetadataNilMap(t *testing.T) {
	ctx := context.Background()
	m := Metadata{Values: nil}

	ctx = WithMetadata(ctx, m)

	got, ok := MetadataFromContext(ctx)
	if !ok {
		t.Fatal("expected metadata in context")
	}
	if got.Values == nil {
		t.Fatal("expected Values map to be initialized")
	}
}
