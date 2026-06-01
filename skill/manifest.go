package skill

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ManifestSchemaV1 is the schema identifier for the v1 agent-package manifest.
const ManifestSchemaV1 = "forge.agent.v1"

// Manifest is the packaged (machine) representation of a Forge agent package. It
// is authored in HCL and compiled to this JSON for distribution inside an OCI
// artifact, so it can be inspected, validated, and consumed without an HCL parser.
type Manifest struct {
	// SchemaVersion identifies the manifest schema (e.g. ManifestSchemaV1).
	SchemaVersion string `json:"schema_version"`
	// Name is the package name.
	Name string `json:"name"`
	// Version is the package version (e.g. an OCI tag like "v1.0.0").
	Version string `json:"version,omitempty"`
	// Description is an optional human-readable summary.
	Description string `json:"description,omitempty"`
	// Skills are the skills the package provides.
	Skills []Spec `json:"skills"`
}

// ParseManifest decodes a manifest from its JSON (machine) representation and
// validates it.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("skill: parse manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// LoadManifest reads and parses a manifest JSON file from path.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("skill: load manifest: %w", err)
	}
	return ParseManifest(data)
}

// Validate checks that the manifest is well-formed: a name is present, the schema
// version is recognized, and every skill is valid with a unique name.
func (m *Manifest) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("skill: manifest name is required")
	}
	if m.SchemaVersion != ManifestSchemaV1 {
		return fmt.Errorf("skill: unsupported manifest schema_version %q (want %q)", m.SchemaVersion, ManifestSchemaV1)
	}
	seen := make(map[string]struct{}, len(m.Skills))
	for _, s := range m.Skills {
		if err := s.Validate(); err != nil {
			return err
		}
		if _, dup := seen[s.Name]; dup {
			return fmt.Errorf("skill: duplicate skill name %q", s.Name)
		}
		seen[s.Name] = struct{}{}
	}
	return nil
}

// GatewayConsumable reports whether the package can be served entirely through
// Forge Core / the gateway without a host process Runner — i.e. all of its
// skills are KindContext. This is the A-layer/B-layer boundary computed at the
// manifest level: a package crosses into "needs a host" the moment it ships a
// KindProcess skill.
func (m *Manifest) GatewayConsumable() bool {
	for _, s := range m.Skills {
		if s.Kind != KindContext {
			return false
		}
	}
	return true
}
