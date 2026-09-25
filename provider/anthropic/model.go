package anthropic

// Model is an Anthropic model identifier.
type Model string

const (
	// ModelClaudeOpus5 is Anthropic's most capable Claude Opus model.
	ModelClaudeOpus5 Model = "claude-opus-5"

	// ModelClaudeSonnet5 is Anthropic's balanced Claude Sonnet model.
	ModelClaudeSonnet5 Model = "claude-sonnet-5"

	// ModelClaudeHaiku45 is Anthropic's fast, low-cost Claude Haiku model.
	ModelClaudeHaiku45 Model = "claude-haiku-4-5"
)
