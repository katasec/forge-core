package registry

import "github.com/katasec/forge-core/tool"

// Registry stores tools and provides lookup by name.
type Registry struct {
	tools map[string]tool.Tool
	order []string
}

// New creates an empty Registry.
func New() *Registry {
	return &Registry{
		tools: make(map[string]tool.Tool),
	}
}

// Register adds one or more tools to the registry.
// Duplicate names overwrite silently (last-write-wins).
func (r *Registry) Register(tools ...tool.Tool) {
	for _, t := range tools {
		name := t.Name()
		if _, exists := r.tools[name]; !exists {
			r.order = append(r.order, name)
		}
		r.tools[name] = t
	}
}

// Get returns a tool by name. Returns false if not found.
func (r *Registry) Get(name string) (tool.Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Definitions returns all registered tools as Definitions,
// in the order they were first registered.
func (r *Registry) Definitions() []tool.Definition {
	defs := make([]tool.Definition, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		defs = append(defs, tool.Definition{
			Name:        t.Name(),
			Description: t.Description(),
			Schema:      t.Schema(),
		})
	}
	return defs
}
