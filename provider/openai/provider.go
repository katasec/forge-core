// Package openai implements forge.Provider using the OpenAI Responses API.
package openai

import (
	"context"
	"net/http"

	"github.com/katasec/forge-core"
)

// Provider implements forge.Provider using the OpenAI Responses API.
type Provider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// New creates an OpenAI provider using the Responses API.
func New(apiKey string, model Model, opts ...Option) *Provider {
	p := &Provider{
		baseURL: "https://api.openai.com/v1",
		apiKey:  apiKey,
		model:   string(model),
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Capabilities describes the OpenAI provider features Forge currently supports.
func (p *Provider) Capabilities() forge.Capabilities {
	return forge.Capabilities{
		Images:     true,
		Usage:      true,
		Production: true,
	}
}

// Generate sends a request to the OpenAI Responses API.
func (p *Provider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
	apiReq, err := p.buildRequest(req)
	if err != nil {
		return nil, err
	}

	apiResp, err := p.sendRequest(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	return providerResponse(apiResp)
}
