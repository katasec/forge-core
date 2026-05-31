package xai

// Citation represents a source reference returned by xAI search tools.
type Citation struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Snippet    string `json:"snippet"`
	Source     string `json:"source"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}

type webSearchConfig struct {
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	ExcludedDomains []string `json:"excluded_domains,omitempty"`
}

type xSearchConfig struct {
	AllowedHandles  []string `json:"allowed_x_handles,omitempty"`
	ExcludedHandles []string `json:"excluded_x_handles,omitempty"`
}
