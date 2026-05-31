// Package xai implements forge.Provider using the xAI Responses API.
//
// This provider supports the modern xAI Responses API with built-in
// server-side tools (web search, X search) and native function calling.
//
// Usage:
//
//	provider := xai.New(apiKey, xai.ModelGrok3Mini, xai.WithWebSearch())
package xai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/katasec/forge-core"
	"github.com/katasec/forge-core/message"
)

// Provider implements forge.Provider using the xAI Responses API.
type Provider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	tools   []requestTool // persistent server-side tools (web_search, x_search)

	mu            sync.Mutex
	lastCitations []Citation
}

// Citation represents a source reference returned by xAI search tools.
type Citation struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Snippet    string `json:"snippet"`
	Source     string `json:"source"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}

// Option configures a Provider.
type Option func(*Provider)

// WebSearchOption configures the web_search tool.
type WebSearchOption func(*webSearchConfig)

// XSearchOption configures the x_search tool.
type XSearchOption func(*xSearchConfig)

type webSearchConfig struct {
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	ExcludedDomains []string `json:"excluded_domains,omitempty"`
}

type xSearchConfig struct {
	AllowedHandles  []string `json:"allowed_x_handles,omitempty"`
	ExcludedHandles []string `json:"excluded_x_handles,omitempty"`
}

// New creates an xAI provider using the Responses API.
func New(apiKey string, model Model, opts ...Option) *Provider {
	p := &Provider{
		baseURL: "https://api.x.ai/v1",
		apiKey:  apiKey,
		model:   string(model),
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Capabilities describes the xAI provider features Forge currently supports.
func (p *Provider) Capabilities() forge.Capabilities {
	return forge.Capabilities{
		Tools:      true,
		Usage:      true,
		Production: true,
	}
}

// WithBaseURL overrides the API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(p *Provider) { p.baseURL = url }
}

// WithWebSearch enables the built-in web search tool.
func WithWebSearch(opts ...WebSearchOption) Option {
	return func(p *Provider) {
		cfg := &webSearchConfig{}
		for _, o := range opts {
			o(cfg)
		}
		t := requestTool{Type: "web_search"}
		if len(cfg.AllowedDomains) > 0 {
			t.AllowedDomains = cfg.AllowedDomains
		}
		if len(cfg.ExcludedDomains) > 0 {
			t.ExcludedDomains = cfg.ExcludedDomains
		}
		p.tools = append(p.tools, t)
	}
}

// WithXSearch enables the built-in X/Twitter search tool.
func WithXSearch(opts ...XSearchOption) Option {
	return func(p *Provider) {
		cfg := &xSearchConfig{}
		for _, o := range opts {
			o(cfg)
		}
		t := requestTool{Type: "x_search"}
		if len(cfg.AllowedHandles) > 0 {
			t.AllowedHandles = cfg.AllowedHandles
		}
		if len(cfg.ExcludedHandles) > 0 {
			t.ExcludedHandles = cfg.ExcludedHandles
		}
		p.tools = append(p.tools, t)
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

// LastCitations returns the citations from the most recent Generate call.
func (p *Provider) LastCitations() []Citation {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastCitations
}

// --- xAI Responses API wire types ---

type request struct {
	Model string        `json:"model"`
	Input []inputItem   `json:"input"`
	Tools []requestTool `json:"tools,omitempty"`
}

type inputItem struct {
	// Message fields
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	// Tool result fields
	Type   string `json:"type,omitempty"` // "function_call_output"
	CallID string `json:"call_id,omitempty"`
	Output string `json:"output,omitempty"`
}

type requestTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	// web_search options
	AllowedDomains  []string `json:"allowed_domains,omitempty"`
	ExcludedDomains []string `json:"excluded_domains,omitempty"`
	// x_search options
	AllowedHandles  []string `json:"allowed_x_handles,omitempty"`
	ExcludedHandles []string `json:"excluded_x_handles,omitempty"`
}

type response struct {
	ID     string        `json:"id"`
	Output []outputItem  `json:"output"`
	Usage  responseUsage `json:"usage"`
}

type outputItem struct {
	Type string `json:"type"`
	// function_call fields
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	// message fields
	Role    string        `json:"role,omitempty"`
	Content []contentItem `json:"content,omitempty"`
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

// --- Conversion helpers ---

// convertMessages converts forge messages to xAI input items.
func convertMessages(msgs []forge.Message, systemPrompt string) []inputItem {
	var items []inputItem

	if systemPrompt != "" {
		items = append(items, inputItem{Role: "system", Content: systemPrompt})
	}

	for _, m := range msgs {
		if m.Role == forge.RoleSystem {
			continue // handled above
		}

		// Tool result messages expand into one input item per result.
		if m.Role == forge.RoleTool && len(m.ToolResults()) > 0 {
			for _, tr := range m.ToolResults() {
				items = append(items, inputItem{
					Type:   "function_call_output",
					CallID: tr.CallID,
					Output: tr.Content,
				})
			}
			continue
		}

		items = append(items, inputItem{
			Role:    string(m.Role),
			Content: m.Text(),
		})
	}

	return items
}

// convertTools converts forge tool definitions to xAI flat tool format.
func convertTools(defs []forge.ToolDefinition) []requestTool {
	tools := make([]requestTool, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, requestTool{
			Type:        "function",
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.Schema.Parameters,
		})
	}
	return tools
}

// parseResponse converts an xAI response to a forge ProviderResponse.
func parseResponse(resp *response) (*forge.ProviderResponse, []Citation) {
	var content string
	var toolCalls []forge.ToolCall
	var citations []Citation

	for _, item := range resp.Output {
		switch item.Type {
		case "function_call":
			toolCalls = append(toolCalls, forge.ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: json.RawMessage(item.Arguments),
			})
		case "message":
			for _, c := range item.Content {
				if c.Type == "output_text" {
					content += c.Text
					// Extract citations from inline annotations.
					for _, a := range c.Annotations {
						if a.Type == "url_citation" {
							citations = append(citations, Citation{
								URL:        a.URL,
								Title:      a.Title,
								Source:     "web",
								StartIndex: a.StartIndex,
								EndIndex:   a.EndIndex,
							})
						}
					}
				}
			}
			// Server-side tool calls (web_search_call, x_search_call, etc.)
			// are auto-executed by xAI, so we don't surface them.
		}
	}

	finishReason := forge.FinishReasonStop
	if len(toolCalls) > 0 {
		finishReason = forge.FinishReasonToolUse
	}

	blocks := []forge.ContentBlock{}
	if content != "" {
		blocks = append(blocks, forge.Text(content))
	}
	for _, call := range toolCalls {
		blocks = append(blocks, message.ToolCall(call))
	}

	return &forge.ProviderResponse{
		Messages:     []forge.Message{{Role: forge.RoleAssistant, Content: blocks}},
		FinishReason: finishReason,
		Usage: forge.TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}, citations
}

// Generate sends a request to the xAI Responses API.
func (p *Provider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
	// Build the input items from forge messages.
	input := convertMessages(req.Messages, req.SystemPrompt)

	// Build tools: merge function tools from the request with persistent server-side tools.
	var tools []requestTool
	tools = append(tools, p.tools...) // server-side tools (web_search, x_search)
	if len(req.Tools) > 0 {
		tools = append(tools, convertTools(req.Tools)...)
	}

	body := request{
		Model: p.model,
		Input: input,
		Tools: tools,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/responses", p.baseURL), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xAI API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	providerResp, citations := parseResponse(&apiResp)

	// Store citations for provider-specific access.
	p.mu.Lock()
	p.lastCitations = citations
	p.mu.Unlock()

	return providerResp, nil
}
