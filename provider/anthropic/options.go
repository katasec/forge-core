package anthropic

import "strings"

// Option configures an AnthropicProvider.
type Option func(*AnthropicProvider)

// WithBaseURL overrides the Anthropic API base URL.
func WithBaseURL(baseURL string) Option {
	return func(p *AnthropicProvider) {
		p.baseURL = strings.TrimRight(baseURL, "/")
	}
}
