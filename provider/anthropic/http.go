package anthropic

import (
	"context"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
)

func (p *AnthropicProvider) sendRequest(ctx context.Context, req anthropicsdk.MessageNewParams) (*anthropicsdk.Message, error) {
	return p.sdkClient.Messages.New(ctx, req)
}
