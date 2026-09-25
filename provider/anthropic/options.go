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

// WithMaxTokens overrides the maximum number of tokens Anthropic may generate
// in a single response. Values below one are ignored.
func WithMaxTokens(maxTokens int) Option {
	return func(p *AnthropicProvider) {
		if maxTokens > 0 {
			p.maxTokens = maxTokens
		}
	}
}
