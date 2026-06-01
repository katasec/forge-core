package skill

import "testing"

const validManifest = `{
  "schema_version": "forge.agent.v1",
  "name": "api-architect",
  "version": "v1.0.0",
  "skills": [
    {"name": "session_start", "kind": "context", "runner": "markdown", "entrypoint": "skills/session_start/skill.md"},
    {"name": "review_api", "kind": "process", "runner": "python", "entrypoint": "skills/review_api/run.py"}
  ]
}`

func TestParseManifest(t *testing.T) {
	m, err := ParseManifest([]byte(validManifest))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if m.Name != "api-architect" {
		t.Errorf("name = %q", m.Name)
	}
	if len(m.Skills) != 2 {
		t.Fatalf("skills = %d, want 2", len(m.Skills))
	}
}

func TestParseManifestErrors(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{name: "bad json", json: `{`},
		{name: "missing name", json: `{"schema_version":"forge.agent.v1","skills":[]}`},
		{name: "wrong schema", json: `{"schema_version":"v2","name":"x","skills":[]}`},
		{name: "invalid skill", json: `{"schema_version":"forge.agent.v1","name":"x","skills":[{"name":"a","kind":"context"}]}`},
		{
			name: "duplicate skill",
			json: `{"schema_version":"forge.agent.v1","name":"x","skills":[
				{"name":"a","kind":"context","runner":"markdown","entrypoint":"a.md"},
				{"name":"a","kind":"context","runner":"markdown","entrypoint":"b.md"}
			]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseManifest([]byte(tt.json)); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestGatewayConsumable(t *testing.T) {
	contextOnly := &Manifest{
		SchemaVersion: ManifestSchemaV1, Name: "x",
		Skills: []Spec{{Name: "a", Kind: KindContext, Runner: "markdown", Entrypoint: "a.md"}},
	}
	if !contextOnly.GatewayConsumable() {
		t.Error("context-only manifest should be gateway-consumable")
	}

	withProcess := &Manifest{
		SchemaVersion: ManifestSchemaV1, Name: "x",
		Skills: []Spec{
			{Name: "a", Kind: KindContext, Runner: "markdown", Entrypoint: "a.md"},
			{Name: "b", Kind: KindProcess, Runner: "python", Entrypoint: "b.py"},
		},
	}
	if withProcess.GatewayConsumable() {
		t.Error("manifest with a process skill must require a host")
	}
}
