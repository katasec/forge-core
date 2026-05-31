package anthropic

import (
	"context"

	"github.com/katasec/forge-core/provider/internal/httpjson"
)

func (p *Provider) sendRequest(ctx context.Context, req request) (*response, error) {
	return httpjson.Post[request, response](ctx, httpjson.Request{
		Client:      p.client,
		URL:         "https://api.anthropic.com/v1/messages",
		Headers:     apiHeaders(p.apiKey),
		ErrorPrefix: "anthropic API error",
	}, req)
}

func apiHeaders(apiKey string) map[string]string {
	return map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}
}
