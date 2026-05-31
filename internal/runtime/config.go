package runtime

import (
	"github.com/katasec/forge-core/memory"
	"github.com/katasec/forge-core/middleware"
	"github.com/katasec/forge-core/provider"
	"github.com/katasec/forge-core/tool"
)

// Config holds the settings for creating an Agent.
type Config struct {
	Provider      provider.Provider
	Tools         []tool.Tool
	Middleware    []middleware.Middleware
	Memory        memory.Store // optional, defaults to in-memory unless DisableMemory is true
	DisableMemory bool         // optional, true means no conversation persistence
	SystemPrompt  string       // optional
	MaxIterations int          // 0 means no limit
	ErrorPolicy   ErrorPolicy  // defaults to ErrorPolicyStop
}
