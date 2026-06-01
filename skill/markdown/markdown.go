// Package markdown provides the built-in context Runner. It is the one Runner
// Forge Core ships, because context skills are enrichment assets that are not
// executed: the runner loads the skill's entrypoint file and returns its content
// for injection into the agent's context. There is no process, filesystem
// mutation, or sandbox involved, so it is safe to run in-process inside Forge
// Core / the gateway.
//
// Process skills (python, shell, ...) are intentionally NOT handled here; they
// require a host-provided Runner that owns the process boundary.
package markdown

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/katasec/forge-core/skill"
)

// Runner loads context (markdown/text) skill assets. It is stateless and safe to
// share across goroutines.
type Runner struct{}

// New returns a context Runner.
func New() *Runner { return &Runner{} }

// Runner reports the runner name this implementation handles.
func (*Runner) Runner() string { return "markdown" }

// Kind reports that this Runner provides context skills.
func (*Runner) Kind() skill.Kind { return skill.KindContext }

// output is the JSON shape returned in Result.Output.
type output struct {
	Content string `json:"content"`
}

// Run loads the skill's entrypoint file relative to in.Root and returns its
// content. It does not execute anything.
func (*Runner) Run(ctx context.Context, spec skill.Spec, in skill.Input) (skill.Result, error) {
	if err := ctx.Err(); err != nil {
		return skill.Result{}, err
	}
	path, err := skill.ResolveEntrypoint(in.Root, spec.Entrypoint)
	if err != nil {
		return skill.Result{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return skill.Result{}, fmt.Errorf("markdown: read skill entrypoint: %w", err)
	}
	out, err := json.Marshal(output{Content: string(data)})
	if err != nil {
		return skill.Result{}, fmt.Errorf("markdown: encode output: %w", err)
	}
	return skill.Success(out), nil
}
