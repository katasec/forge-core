package xai

import "encoding/json"

type request struct {
	Model string        `json:"model"`
	Input []inputItem   `json:"input"`
	Tools []requestTool `json:"tools,omitempty"`
}

type inputItem struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Type    string `json:"type,omitempty"`
	CallID  string `json:"call_id,omitempty"`
	Output  string `json:"output,omitempty"`
}

type requestTool struct {
	Type            string          `json:"type"`
	Name            string          `json:"name,omitempty"`
	Description     string          `json:"description,omitempty"`
	Parameters      json.RawMessage `json:"parameters,omitempty"`
	AllowedDomains  []string        `json:"allowed_domains,omitempty"`
	ExcludedDomains []string        `json:"excluded_domains,omitempty"`
	AllowedHandles  []string        `json:"allowed_x_handles,omitempty"`
	ExcludedHandles []string        `json:"excluded_x_handles,omitempty"`
}

type response struct {
	ID     string        `json:"id"`
	Output []outputItem  `json:"output"`
	Usage  responseUsage `json:"usage"`
}

type outputItem struct {
	Type      string        `json:"type"`
	Name      string        `json:"name,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Role      string        `json:"role,omitempty"`
	Content   []contentItem `json:"content,omitempty"`
}

type contentItem struct {
	Type        string       `json:"type"`
	Text        string       `json:"text"`
	Annotations []annotation `json:"annotations,omitempty"`
}

type annotation struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}

type responseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
