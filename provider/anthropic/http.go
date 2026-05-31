package anthropic

import (
	"context"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
)

func (p *Provider) sendRequest(ctx context.Context, req anthropicsdk.MessageNewParams) (*anthropicsdk.Message, error) {
	return p.sdkClient.Messages.New(ctx, req)
}
