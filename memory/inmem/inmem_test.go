package inmem

import (
	"context"
	"testing"

	"github.com/katasec/forge-core/message"
)

func TestStoreLoadEmpty(t *testing.T) {
	s := New()
	msgs, err := s.Load(context.Background(), "conv-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil for empty conversation, got %v", msgs)
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	s := New()
	ctx := context.Background()

	messages := []message.Message{
		{ID: "1", Role: message.RoleUser, Content: []message.ContentBlock{message.Text("hello")}},
		{ID: "2", Role: message.RoleAssistant, Content: []message.ContentBlock{message.Text("hi")}},
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

func TestStoreSaveReplaces(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Save(ctx, "conv-1", []message.Message{{ID: "1", Content: []message.ContentBlock{message.Text("first")}}}); err != nil {
		t.Fatalf("first Save error: %v", err)
	}
	if err := s.Save(ctx, "conv-1", []message.Message{{ID: "2", Content: []message.ContentBlock{message.Text("second")}}}); err != nil {
		t.Fatalf("second Save error: %v", err)
	}

	loaded, err := s.Load(ctx, "conv-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("got %d messages, want 1", len(loaded))
	}
	if loaded[0].Text() != "second" {
		t.Errorf("Content = %q, want %q", loaded[0].Text(), "second")
	}
}

func TestStoreClear(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Save(ctx, "conv-1", []message.Message{{ID: "1", Content: []message.ContentBlock{message.Text("hello")}}}); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	if err := s.Clear(ctx, "conv-1"); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	loaded, err := s.Load(ctx, "conv-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil after clear, got %v", loaded)
	}
}

func TestStoreReturnsCopy(t *testing.T) {
	s := New()
	ctx := context.Background()

	if err := s.Save(ctx, "conv-1", []message.Message{{ID: "1", Content: []message.ContentBlock{message.Text("original")}}}); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := s.Load(ctx, "conv-1")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	loaded[0].Content[0].Text = "mutated"

	reloaded, err := s.Load(ctx, "conv-1")
	if err != nil {
		t.Fatalf("Reload error: %v", err)
	}
	if reloaded[0].Text() != "original" {
		t.Errorf("Content = %q, want %q (store should return copies)", reloaded[0].Text(), "original")
	}
}
