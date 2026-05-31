package openai

import (
	"context"

	"github.com/openai/openai-go/v3/responses"
)

func (p *Provider) sendRequest(ctx context.Context, req responses.ResponseNewParams) (*responses.Response, error) {
	return p.sdkClient.Responses.New(ctx, req)
}
