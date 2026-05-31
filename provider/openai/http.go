package openai

import (
	"context"

	"github.com/openai/openai-go/v3/responses"
)

// sendRequest sends a prepared OpenAI Responses request through the SDK client.
func (p *OpenAIProvider) sendRequest(ctx context.Context, req responses.ResponseNewParams) (*responses.Response, error) {
	return p.sdkClient.Responses.New(ctx, req)
}
