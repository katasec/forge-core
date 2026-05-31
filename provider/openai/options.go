package openai

import "strings"

// Option configures an OpenAIProvider.
type Option func(*OpenAIProvider)

// WithBaseURL overrides the OpenAI API base URL.
func WithBaseURL(baseURL string) Option {
	return func(p *OpenAIProvider) {
		p.baseURL = strings.TrimRight(baseURL, "/")
	}
}
