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
	"context"
	"net/http"
	"sync"

	"github.com/katasec/forge-core"
)

// Provider implements forge.Provider using the xAI Responses API.
type Provider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	tools   []requestTool

	mu            sync.Mutex
	lastCitations []Citation
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

// LastCitations returns the citations from the most recent Generate call.
func (p *Provider) LastCitations() []Citation {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastCitations
}

// Generate sends a request to the xAI Responses API.
func (p *Provider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
	apiReq := request{
		Model: p.model,
		Input: convertMessages(req.Messages, req.SystemPrompt),
		Tools: p.requestTools(req.Tools),
	}

	apiResp, err := p.sendRequest(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	providerResp, citations := providerResponse(apiResp)
	p.storeCitations(citations)
	return providerResp, nil
}

func (p *Provider) requestTools(defs []forge.ToolDefinition) []requestTool {
	var tools []requestTool
	tools = append(tools, p.tools...)
	if len(defs) > 0 {
		tools = append(tools, convertTools(defs)...)
	}
	return tools
}

func (p *Provider) storeCitations(citations []Citation) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastCitations = citations
}
