// Package tests holds end-to-end smoke tests that exercise the public Forge Core
// contract across packages, using a real on-disk agent package under testdata/.
package tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/katasec/forge-core/skill"
	"github.com/katasec/forge-core/skill/markdown"
)

const packageRoot = "testdata/simple-agent"

// TestContextSkillEndToEnd proves the basic contract: load a manifest with a
// single context skill, register the markdown runner, resolve it, execute it,
// and get the markdown file's content back.
func TestContextSkillEndToEnd(t *testing.T) {
	// 1. Load the manifest.
	m, err := skill.LoadManifest(filepath.Join(packageRoot, "forge.json"))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if len(m.Skills) != 1 {
		t.Fatalf("skills = %d, want 1", len(m.Skills))
	}
	if !m.GatewayConsumable() {
		t.Error("a context-only package should be gateway-consumable")
	}
	spec := m.Skills[0]

	// 2. Register the built-in context runner.
	reg := skill.NewRegistry(markdown.New())

	// 3. Resolve: a matching runner is available, so it should execute.
	if got := reg.Resolve(spec); got != skill.OutcomeExecute {
		t.Fatalf("Resolve() = %q, want %q", got, skill.OutcomeExecute)
	}

	// 4. Execute / load the skill.
	res, err := reg.Execute(context.Background(), spec, skill.Input{Root: packageRoot})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if res.Status != skill.StatusSuccess {
		t.Fatalf("status = %q, want %q", res.Status, skill.StatusSuccess)
	}

	// 5. Verify the returned content matches the file on disk.
	var out struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(res.Output, &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(packageRoot, spec.Entrypoint))
	if err != nil {
		t.Fatal(err)
	}
	if out.Content != string(want) {
		t.Errorf("content mismatch:\n got: %q\nwant: %q", out.Content, want)
	}
}

// TestTraversalRejected proves a skill cannot read outside its package root.
func TestTraversalRejected(t *testing.T) {
	reg := skill.NewRegistry(markdown.New())
	spec := skill.Spec{
		Name:       "evil",
		Kind:       skill.KindContext,
		Runner:     "markdown",
		Entrypoint: "../../../../etc/passwd",
	}
	if _, err := reg.Execute(context.Background(), spec, skill.Input{Root: packageRoot}); err == nil {
		t.Fatal("expected path-traversal entrypoint to be rejected, got nil error")
	}
}

// TestUnknownRunnerExposes proves an unsupported runner resolves to expose
// instead of crashing — the host runtime can then decide what to do.
func TestUnknownRunnerExposes(t *testing.T) {
	reg := skill.NewRegistry(markdown.New())
	spec := skill.Spec{
		Name:       "review_api",
		Kind:       skill.KindProcess,
		Runner:     "python",
		Entrypoint: "skills/review_api/run.py",
	}
	if got := reg.Resolve(spec); got != skill.OutcomeExpose {
		t.Errorf("Resolve() = %q, want %q", got, skill.OutcomeExpose)
	}
}
