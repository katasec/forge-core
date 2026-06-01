package skill

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		wantErr bool
	}{
		{
			name: "valid context",
			spec: Spec{Name: "session_start", Kind: KindContext, Runner: "markdown", Entrypoint: "skills/s/skill.md"},
		},
		{
			name: "valid process",
			spec: Spec{Name: "review_api", Kind: KindProcess, Runner: "python", Entrypoint: "skills/r/run.py"},
		},
		{
			name:    "missing name",
			spec:    Spec{Kind: KindContext, Runner: "markdown", Entrypoint: "x.md"},
			wantErr: true,
		},
		{
			name:    "invalid kind",
			spec:    Spec{Name: "x", Kind: "wasm", Runner: "markdown", Entrypoint: "x.md"},
			wantErr: true,
		},
		{
			name:    "missing runner",
			spec:    Spec{Name: "x", Kind: KindContext, Entrypoint: "x.md"},
			wantErr: true,
		},
		{
			name:    "missing entrypoint",
			spec:    Spec{Name: "x", Kind: KindContext, Runner: "markdown"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSuccessNormalizesEmptyOutput(t *testing.T) {
	r := Success(nil)
	if r.Schema != ResultSchemaV1 {
		t.Errorf("schema = %q, want %q", r.Schema, ResultSchemaV1)
	}
	if r.Status != StatusSuccess {
		t.Errorf("status = %q, want %q", r.Status, StatusSuccess)
	}
	if string(r.Output) != "{}" {
		t.Errorf("output = %q, want {}", r.Output)
	}
	if r.Error != nil {
		t.Errorf("error = %v, want nil", r.Error)
	}
}

func TestResultEnvelopeShape(t *testing.T) {
	data, err := json.Marshal(Success(json.RawMessage(`{"k":"v"}`)))
	if err != nil {
		t.Fatal(err)
	}
	// The envelope must carry all four fields, with output and error always present.
	var got map[string]json.RawMessage
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"schema", "status", "output", "error"} {
		if _, ok := got[k]; !ok {
			t.Errorf("envelope missing field %q: %s", k, data)
		}
	}
	if string(got["error"]) != "null" {
		t.Errorf("error = %s, want null", got["error"])
	}
}

func TestFailure(t *testing.T) {
	r := Failure("boom")
	if r.Status != StatusError {
		t.Errorf("status = %q, want %q", r.Status, StatusError)
	}
	if r.Error == nil || *r.Error != "boom" {
		t.Errorf("error = %v, want boom", r.Error)
	}
	if string(r.Output) != "{}" {
		t.Errorf("output = %q, want {}", r.Output)
	}
}

func TestResolveEntrypoint(t *testing.T) {
	root := "/pkg/root"
	tests := []struct {
		name       string
		entrypoint string
		want       string
		wantErr    bool
	}{
		{name: "simple", entrypoint: "skills/a/run.py", want: filepath.Join(root, "skills/a/run.py")},
		{name: "dot prefix", entrypoint: "./skills/a.md", want: filepath.Join(root, "skills/a.md")},
		{name: "empty", entrypoint: "", wantErr: true},
		{name: "absolute", entrypoint: "/etc/passwd", wantErr: true},
		{name: "traversal", entrypoint: "../../etc/passwd", wantErr: true},
		{name: "sneaky traversal", entrypoint: "skills/../../secret", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveEntrypoint(root, tt.entrypoint)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
