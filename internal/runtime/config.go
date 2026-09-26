package runtime

import (
	"github.com/katasec/kiln/memory"
	"github.com/katasec/kiln/middleware"
	"github.com/katasec/kiln/provider"
	"github.com/katasec/kiln/tool"
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
