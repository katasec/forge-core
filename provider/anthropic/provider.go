// Package anthropic implements forge.Provider using the Anthropic Messages API.
package anthropic

import (
	"context"
	"net/http"

	"github.com/katasec/forge-core"
)

// Provider implements forge.Provider using the Anthropic Messages API.
type Provider struct {
	apiKey string
	model  string
	client *http.Client
}

// New creates an Anthropic provider for the given API key and model.
func New(apiKey, model string) *Provider {
	return &Provider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Capabilities describes the Anthropic provider features Forge currently supports.
func (p *Provider) Capabilities() forge.Capabilities {
	return forge.Capabilities{
		Usage:      true,
		Production: true,
	}
}

// Generate sends a request to the Anthropic Messages API.
func (p *Provider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
	apiReq := p.buildRequest(req)

	apiResp, err := p.sendRequest(ctx, apiReq)
	if err != nil {
		return nil, err
	}

	return providerResponse(apiResp), nil
}
