package forge

import (
	"context"
	"testing"
)

func TestInMemoryStoreLoadEmpty(t *testing.T) {
	s := NewInMemoryStore()
	msgs, err := s.Load(context.Background(), "conv-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil for empty conversation, got %v", msgs)
	}
}

func TestInMemoryStoreSaveAndLoad(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	messages := []Message{
		{ID: "1", Role: RoleUser, Content: []ContentBlock{Text("hello")}},
		{ID: "2", Role: RoleAssistant, Content: []ContentBlock{Text("hi")}},
	}

	if err := s.Save(ctx, "conv-1", messages); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := s.Load(ctx, "conv-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("got %d messages, want 2", len(loaded))
	}
	if loaded[0].Text() != "hello" {
		t.Errorf("loaded[0].Text() = %q, want %q", loaded[0].Text(), "hello")
	}
}

func TestInMemoryStoreSaveReplaces(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	s.Save(ctx, "conv-1", []Message{{ID: "1", Content: []ContentBlock{Text("first")}}})
	s.Save(ctx, "conv-1", []Message{{ID: "2", Content: []ContentBlock{Text("second")}}})

	loaded, _ := s.Load(ctx, "conv-1")
	if len(loaded) != 1 {
		t.Fatalf("got %d messages, want 1", len(loaded))
	}
	if loaded[0].Text() != "second" {
		t.Errorf("Content = %q, want %q", loaded[0].Text(), "second")
	}
}

func TestInMemoryStoreClear(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	s.Save(ctx, "conv-1", []Message{{ID: "1", Content: []ContentBlock{Text("hello")}}})

	if err := s.Clear(ctx, "conv-1"); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	loaded, _ := s.Load(ctx, "conv-1")
	if loaded != nil {
		t.Errorf("expected nil after clear, got %v", loaded)
	}
}

func TestInMemoryStoreReturnsCopy(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	s.Save(ctx, "conv-1", []Message{{ID: "1", Content: []ContentBlock{Text("original")}}})

	loaded, _ := s.Load(ctx, "conv-1")
	loaded[0].Content[0].Text = "mutated"

	reloaded, _ := s.Load(ctx, "conv-1")
	if reloaded[0].Text() != "original" {
		t.Errorf("Content = %q, want %q (store should return copies)", reloaded[0].Text(), "original")
	}
}
