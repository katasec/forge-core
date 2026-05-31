package xai

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

func (p *XAIProvider) sendRequest(ctx context.Context, req responses.ResponseNewParams, tools []requestTool) (*response, error) {
	opts := requestOptions(tools)
	apiResp, err := p.sdk.Responses.New(ctx, req, opts...)
	if err != nil {
		return nil, err
	}

	var resp response
	if err := json.Unmarshal([]byte(apiResp.RawJSON()), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func requestOptions(tools []requestTool) []option.RequestOption {
	if len(tools) == 0 {
		return nil
	}
	return []option.RequestOption{option.WithJSONSet("tools", tools)}
}
