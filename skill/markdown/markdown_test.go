package markdown

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/katasec/forge-core/skill"
)

func TestRunLoadsContent(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "skills", "session_start")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := "# Session Start\nDo the thing."
	if err := os.WriteFile(filepath.Join(dir, "skill.md"), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New()
	spec := skill.Spec{
		Name:       "session_start",
		Kind:       skill.KindContext,
		Runner:     "markdown",
		Entrypoint: "skills/session_start/skill.md",
	}
	res, err := r.Run(context.Background(), spec, skill.Input{Root: root})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if res.Status != skill.StatusSuccess {
		t.Fatalf("status = %q", res.Status)
	}

	var out struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(res.Output, &out); err != nil {
		t.Fatal(err)
	}
	if out.Content != want {
		t.Errorf("content = %q, want %q", out.Content, want)
	}
}

func TestRunRejectsTraversal(t *testing.T) {
	r := New()
	spec := skill.Spec{Name: "x", Kind: skill.KindContext, Runner: "markdown", Entrypoint: "../../etc/passwd"}
	if _, err := r.Run(context.Background(), spec, skill.Input{Root: t.TempDir()}); err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestRunMissingFile(t *testing.T) {
	r := New()
	spec := skill.Spec{Name: "x", Kind: skill.KindContext, Runner: "markdown", Entrypoint: "nope.md"}
	if _, err := r.Run(context.Background(), spec, skill.Input{Root: t.TempDir()}); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestKindAndName(t *testing.T) {
	r := New()
	if r.Runner() != "markdown" {
		t.Errorf("Runner() = %q", r.Runner())
	}
	if r.Kind() != skill.KindContext {
		t.Errorf("Kind() = %q", r.Kind())
	}
	// Compile-time check that *Runner satisfies skill.Runner.
	var _ skill.Runner = New()
}
