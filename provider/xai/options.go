package xai

// Option configures an XAIProvider.
type Option func(*XAIProvider)

// WebSearchOption configures the web_search tool.
type WebSearchOption func(*webSearchConfig)

// XSearchOption configures the x_search tool.
type XSearchOption func(*xSearchConfig)

// WithBaseURL overrides the API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(p *XAIProvider) { p.baseURL = url }
}

// WithWebSearch enables the built-in web search tool.
func WithWebSearch(opts ...WebSearchOption) Option {
	return func(p *XAIProvider) {
		cfg := &webSearchConfig{}
		for _, o := range opts {
			o(cfg)
		}
		p.tools = append(p.tools, webSearchTool(cfg))
	}
}

// WithXSearch enables the built-in X/Twitter search tool.
func WithXSearch(opts ...XSearchOption) Option {
	return func(p *XAIProvider) {
		cfg := &xSearchConfig{}
		for _, o := range opts {
			o(cfg)
		}
		p.tools = append(p.tools, xSearchTool(cfg))
	}
}

// AllowedDomains restricts web search to the specified domains.
func AllowedDomains(domains ...string) WebSearchOption {
	return func(c *webSearchConfig) { c.AllowedDomains = domains }
}

// ExcludedDomains excludes the specified domains from web search.
func ExcludedDomains(domains ...string) WebSearchOption {
	return func(c *webSearchConfig) { c.ExcludedDomains = domains }
}

// AllowedHandles restricts X search to the specified handles.
func AllowedHandles(handles ...string) XSearchOption {
	return func(c *xSearchConfig) { c.AllowedHandles = handles }
}

// ExcludedHandles excludes the specified handles from X search.
func ExcludedHandles(handles ...string) XSearchOption {
	return func(c *xSearchConfig) { c.ExcludedHandles = handles }
}

func webSearchTool(cfg *webSearchConfig) requestTool {
	t := requestTool{Type: "web_search"}
	if len(cfg.AllowedDomains) > 0 {
		t.AllowedDomains = cfg.AllowedDomains
	}
	if len(cfg.ExcludedDomains) > 0 {
		t.ExcludedDomains = cfg.ExcludedDomains
	}
	return t
}

func xSearchTool(cfg *xSearchConfig) requestTool {
	t := requestTool{Type: "x_search"}
	if len(cfg.AllowedHandles) > 0 {
		t.AllowedHandles = cfg.AllowedHandles
	}
	if len(cfg.ExcludedHandles) > 0 {
		t.ExcludedHandles = cfg.ExcludedHandles
	}
	return t
}
