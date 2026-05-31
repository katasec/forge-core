package openai

import (
	"context"

	"github.com/katasec/forge-core/provider/internal/httpjson"
)

func (p *Provider) sendRequest(ctx context.Context, req request) (*response, error) {
	return httpjson.Post[request, response](ctx, httpjson.Request{
		Client:      p.client,
		URL:         p.baseURL + "/responses",
		Headers:     bearerHeaders(p.apiKey),
		ErrorPrefix: "openai API error",
	}, req)
}

func bearerHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
	}
}
