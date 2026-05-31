package forge

import "github.com/katasec/forge-core/tool/registry"

type ToolRegistry = registry.Registry

func NewToolRegistry() *ToolRegistry {
	return registry.New()
}
