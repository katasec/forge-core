// Package anthropic implements forge.Provider using the Anthropic Messages API.
package anthropic

import (
	"context"
	"net/http"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/katasec/forge-core"
)

// Provider implements forge.Provider using the Anthropic Messages API.
type Provider struct {
	baseURL   string
	apiKey    string
	model     string
	client    *http.Client
	sdkClient anthropicsdk.Client
}

// New creates an Anthropic provider for the given API key and model.
func New(apiKey, model string, opts ...Option) *Provider {
	p := &Provider{
		baseURL: "https://api.anthropic.com",
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	p.sdkClient = p.newSDKClient()
	return p
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

func (p *Provider) newSDKClient() anthropicsdk.Client {
	return anthropicsdk.NewClient(
		option.WithAPIKey(p.apiKey),
		option.WithBaseURL(p.baseURL),
		option.WithHTTPClient(p.client),
	)
}
