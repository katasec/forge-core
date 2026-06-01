package skill

import (
	"context"
	"fmt"
)

// Outcome is how a skill can be handled given the available Runners. The first
// two are mechanism, computed by Resolve; OutcomeRefuse is a policy decision made
// by the gateway/host (e.g. disallowed permissions) and is part of the contract
// vocabulary, not a value Resolve returns on its own.
type Outcome string

const (
	// OutcomeExecute means a matching Runner is available and the skill can run.
	OutcomeExecute Outcome = "execute"
	// OutcomeExpose means no matching Runner is available, but the skill's
	// metadata and assets can be surfaced to a host runtime to decide what to do
	// (e.g. Claude Code reads run.py and executes it via its own tools).
	OutcomeExpose Outcome = "expose"
	// OutcomeRefuse means the skill must not be run or exposed. This is a policy
	// outcome decided by the gateway/host, never by Resolve.
	OutcomeRefuse Outcome = "refuse"
)

// Registry holds the Runners available to interpret skills, keyed by runner name.
type Registry struct {
	runners map[string]Runner
}

// NewRegistry builds a Registry from the given Runners. A later registration of
// the same runner name overrides an earlier one.
func NewRegistry(runners ...Runner) *Registry {
	r := &Registry{runners: make(map[string]Runner, len(runners))}
	for _, run := range runners {
		r.Register(run)
	}
	return r
}

// Register adds run to the registry under its Runner() name.
func (r *Registry) Register(run Runner) {
	if r.runners == nil {
		r.runners = make(map[string]Runner)
	}
	r.runners[run.Runner()] = run
}

// Get returns the Runner registered under name.
func (r *Registry) Get(name string) (Runner, bool) {
	run, ok := r.runners[name]
	return run, ok
}

// Resolve reports whether spec can be executed with the registered Runners. It
// returns OutcomeExecute when a Runner matching both the spec's runner name and
// Kind is available, otherwise OutcomeExpose. It never returns OutcomeRefuse —
// refusal is a policy decision left to the gateway/host.
func (r *Registry) Resolve(spec Spec) Outcome {
	if run, ok := r.runners[spec.Runner]; ok && run.Kind() == spec.Kind {
		return OutcomeExecute
	}
	return OutcomeExpose
}

// Execute resolves and runs spec. It returns an error if no matching Runner is
// available (the caller should fall back to exposing the skill); a skill that ran
// but reported failure is conveyed via Result.Status, not an error.
func (r *Registry) Execute(ctx context.Context, spec Spec, in Input) (Result, error) {
	if r.Resolve(spec) != OutcomeExecute {
		return Result{}, fmt.Errorf("skill %q: no %s runner %q available", spec.Name, spec.Kind, spec.Runner)
	}
	return r.runners[spec.Runner].Run(ctx, spec, in)
}
