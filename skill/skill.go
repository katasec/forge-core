// Package skill defines the contract for packaged skills: the unit Forge
// distributes inside an agent package.
//
// Forge Core is the interpreter and contract definer, NOT a sandbox. It parses
// the skill manifest, models skills, and defines the Runner interface used to
// execute them. It ships only the in-process context Runner (see skill/markdown);
// process skills require a host-provided Runner that owns the process boundary
// and enforces isolation.
//
// Skills come in two kinds with opposite execution semantics:
//
//   - KindContext skills are enrichment assets. They are not executed; their
//     content is loaded and injected into the agent's context. They are safe to
//     run in-process inside Forge Core / the gateway.
//   - KindProcess skills execute as an external process (python, shell, ...).
//     Forge Core never runs them by default; they require a host-provided Runner.
//
// A package whose skills are all KindContext is gateway-consumable; the moment a
// package ships a KindProcess skill it requires a host with an execution-capable
// Runner. See Manifest.GatewayConsumable.
package skill

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Kind classifies how a skill is consumed.
type Kind string

const (
	// KindContext skills are enrichment assets: not executed, injected as context.
	// Safe to run in-process inside Forge Core / the gateway.
	KindContext Kind = "context"
	// KindProcess skills execute as an external process. They require a
	// host-provided Runner with explicit execution authority; Forge Core never
	// runs them by default.
	KindProcess Kind = "process"
)

// Valid reports whether k is a recognized kind.
func (k Kind) Valid() bool {
	return k == KindContext || k == KindProcess
}

// Spec is the manifest description of a single skill. It is authored in HCL and
// compiled to JSON for distribution inside an OCI artifact; this struct is the
// JSON (machine) representation.
type Spec struct {
	// Name is the unique identifier of the skill within its package.
	Name string `json:"name"`
	// Kind classifies execution semantics (context vs process).
	Kind Kind `json:"kind"`
	// Runner names the Runner implementation that handles this skill
	// (e.g. "markdown", "python"). It must be matched by a registered Runner
	// of the same Kind for the skill to be executable.
	Runner string `json:"runner"`
	// Entrypoint is the package-relative path to the skill's asset or script.
	Entrypoint string `json:"entrypoint"`
	// Description is an optional human-readable summary.
	Description string `json:"description,omitempty"`
	// Inputs is an optional JSON Schema describing the skill's accepted input.
	Inputs json.RawMessage `json:"inputs,omitempty"`
	// Outputs is an optional JSON Schema describing the skill's output.
	Outputs json.RawMessage `json:"outputs,omitempty"`
	// Permissions are capability requests declared by the skill (e.g. "net",
	// "fs:write"). Forge Core surfaces and validates these; it does NOT enforce
	// them. Enforcement belongs to the Runner/gateway/host that owns the process
	// boundary.
	Permissions []string `json:"permissions,omitempty"`
}

// Validate checks that the spec is well-formed. It does not check that a Runner
// is available — that is a resolution-time concern (see Registry.Resolve).
func (s Spec) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("skill: name is required")
	}
	if !s.Kind.Valid() {
		return fmt.Errorf("skill %q: invalid kind %q (want %q or %q)", s.Name, s.Kind, KindContext, KindProcess)
	}
	if strings.TrimSpace(s.Runner) == "" {
		return fmt.Errorf("skill %q: runner is required", s.Name)
	}
	if strings.TrimSpace(s.Entrypoint) == "" {
		return fmt.Errorf("skill %q: entrypoint is required", s.Name)
	}
	return nil
}

// Input is passed to a Runner when executing a skill.
type Input struct {
	// Root is the on-disk package root used to resolve Spec.Entrypoint. It is
	// runtime state, not part of the serialized contract.
	Root string `json:"-"`
	// Args is the JSON-encoded input to the skill, conforming to Spec.Inputs.
	Args json.RawMessage `json:"args,omitempty"`
}

// Status is the outcome status of a skill execution.
type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
)

// ResultSchemaV1 is the schema identifier for the v1 result envelope.
const ResultSchemaV1 = "forge.skill.result.v1"

// Result is the structured output of a skill execution. Runners emit it as JSON
// on stdout; stderr is reserved for logs.
type Result struct {
	Schema string          `json:"schema"`
	Status Status          `json:"status"`
	Output json.RawMessage `json:"output"`
	Error  *string         `json:"error"`
}

// Success builds a successful Result. A nil/empty output is normalized to "{}".
func Success(output json.RawMessage) Result {
	if len(output) == 0 {
		output = json.RawMessage("{}")
	}
	return Result{Schema: ResultSchemaV1, Status: StatusSuccess, Output: output}
}

// Failure builds an error Result carrying msg.
func Failure(msg string) Result {
	return Result{Schema: ResultSchemaV1, Status: StatusError, Output: json.RawMessage("{}"), Error: &msg}
}

// Runner executes skills of a particular runner type. Forge Core ships only the
// in-process context Runner (skill/markdown); process Runners must be supplied by
// the host/gateway, which owns the process boundary and enforces permissions.
type Runner interface {
	// Runner reports the runner name this implementation handles (e.g. "markdown").
	Runner() string
	// Kind reports the execution kind this Runner provides.
	Kind() Kind
	// Run executes the skill and returns a structured Result. A returned error
	// indicates the Runner itself failed; a skill that ran but reported failure
	// is conveyed via Result.Status.
	Run(ctx context.Context, spec Spec, in Input) (Result, error)
}

// ResolveEntrypoint joins a package-relative entrypoint to root, rejecting
// absolute paths and any path that escapes root. Runners should use this to
// resolve Spec.Entrypoint safely.
func ResolveEntrypoint(root, entrypoint string) (string, error) {
	if strings.TrimSpace(entrypoint) == "" {
		return "", errors.New("skill: entrypoint is empty")
	}
	clean := filepath.Clean(entrypoint)
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("skill: entrypoint must be package-relative: %q", entrypoint)
	}
	full := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("skill: entrypoint escapes package root: %q", entrypoint)
	}
	return full, nil
}
