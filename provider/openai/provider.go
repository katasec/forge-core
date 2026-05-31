// Package openai implements forge.Provider using the OpenAI Responses API.
package openai

import (
	"context"
	"net/http"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/katasec/forge-core"
)

// OpenAIProvider implements forge.Provider using the OpenAI Responses API.
type OpenAIProvider struct {
	baseURL   string
	apiKey    string
	model     string
	client    *http.Client
	sdkClient openaisdk.Client
}

// New creates an OpenAI provider using the Responses API.
func New(apiKey string, model Model, opts ...Option) *OpenAIProvider {
	p := &OpenAIProvider{
		baseURL: "https://api.openai.com/v1",
		apiKey:  apiKey,
		model:   string(model),
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	p.sdkClient = p.newSDKClient()
	return p
}

// Capabilities describes the OpenAI provider features Forge currently supports.
func (p *OpenAIProvider) Capabilities() forge.Capabilities {
	return forge.Capabilities{
		Images:     true,
		Usage:      true,
		Production: true,
	}
}

// Generate sends a request to the OpenAI Responses API.
func (p *OpenAIProvider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
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

// newSDKClient builds the OpenAI SDK client from provider configuration.
func (p *OpenAIProvider) newSDKClient() openaisdk.Client {
	return openaisdk.NewClient(
		option.WithAPIKey(p.apiKey),
		option.WithBaseURL(p.baseURL),
		option.WithHTTPClient(p.client),
	)
}
