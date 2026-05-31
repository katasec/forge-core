package anthropic

import "strings"

// Option configures a Provider.
type Option func(*Provider)

// WithBaseURL overrides the Anthropic API base URL.
func WithBaseURL(baseURL string) Option {
	return func(p *Provider) {
		p.baseURL = strings.TrimRight(baseURL, "/")
	}
}
