package skill

import (
	"context"
	"encoding/json"
	"testing"
)

// stubRunner is a test Runner that records invocation and returns a canned result.
type stubRunner struct {
	name   string
	kind   Kind
	called bool
}

func (s *stubRunner) Runner() string { return s.name }
func (s *stubRunner) Kind() Kind     { return s.kind }
func (s *stubRunner) Run(ctx context.Context, spec Spec, in Input) (Result, error) {
	s.called = true
	return Success(json.RawMessage(`{"ran":true}`)), nil
}

func TestResolve(t *testing.T) {
	reg := NewRegistry(&stubRunner{name: "markdown", kind: KindContext})

	tests := []struct {
		name string
		spec Spec
		want Outcome
	}{
		{
			name: "matching runner and kind executes",
			spec: Spec{Name: "a", Kind: KindContext, Runner: "markdown", Entrypoint: "a.md"},
			want: OutcomeExecute,
		},
		{
			name: "unknown runner exposes",
			spec: Spec{Name: "b", Kind: KindProcess, Runner: "python", Entrypoint: "b.py"},
			want: OutcomeExpose,
		},
		{
			name: "known runner wrong kind exposes",
			spec: Spec{Name: "c", Kind: KindProcess, Runner: "markdown", Entrypoint: "c.py"},
			want: OutcomeExpose,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reg.Resolve(tt.spec); got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExecuteRunsMatchingRunner(t *testing.T) {
	run := &stubRunner{name: "markdown", kind: KindContext}
	reg := NewRegistry(run)
	spec := Spec{Name: "a", Kind: KindContext, Runner: "markdown", Entrypoint: "a.md"}

	res, err := reg.Execute(context.Background(), spec, Input{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !run.called {
		t.Error("runner was not invoked")
	}
	if res.Status != StatusSuccess {
		t.Errorf("status = %q, want %q", res.Status, StatusSuccess)
	}
}

func TestExecuteErrorsWithoutRunner(t *testing.T) {
	reg := NewRegistry() // empty
	spec := Spec{Name: "b", Kind: KindProcess, Runner: "python", Entrypoint: "b.py"}

	if _, err := reg.Execute(context.Background(), spec, Input{}); err == nil {
		t.Fatal("Execute() expected error for missing runner, got nil")
	}
}

func TestRegisterOverrides(t *testing.T) {
	first := &stubRunner{name: "markdown", kind: KindContext}
	second := &stubRunner{name: "markdown", kind: KindContext}
	reg := NewRegistry(first)
	reg.Register(second)

	got, ok := reg.Get("markdown")
	if !ok {
		t.Fatal("expected markdown runner")
	}
	if got != second {
		t.Error("Register did not override the earlier runner")
	}
}
