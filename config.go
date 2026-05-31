package forge

// Config holds the settings for creating an Agent.
type Config struct {
	Provider      Provider
	Tools         []Tool
	Middleware    []Middleware
	Memory        MemoryStore // optional, defaults to in-memory unless DisableMemory is true
	DisableMemory bool        // optional, true means no conversation persistence
	SystemPrompt  string      // optional
	MaxIterations int         // 0 means no limit
	ErrorPolicy   ErrorPolicy // defaults to ErrorPolicyStop
}
