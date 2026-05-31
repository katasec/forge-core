package anthropic

import "github.com/katasec/forge-core"

func (p *Provider) buildRequest(req forge.ProviderRequest) request {
	return request{
		Model:     p.model,
		MaxTokens: 1024,
		System:    req.SystemPrompt,
		Messages:  convertMessages(req.Messages),
	}
}

func convertMessages(messages []forge.Message) []message {
	out := make([]message, 0, len(messages))
	for _, m := range messages {
		if m.Role == forge.RoleSystem {
			continue
		}
		out = append(out, message{
			Role:    string(m.Role),
			Content: m.Text(),
		})
	}
	return out
}

func providerResponse(apiResp *response) *forge.ProviderResponse {
	return &forge.ProviderResponse{
		Messages:     []forge.Message{forge.AssistantText(textContent(apiResp.Content))},
		FinishReason: finishReason(apiResp.StopReason),
		Usage: forge.TokenUsage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
		},
	}
}

func textContent(content []content) string {
	for _, c := range content {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}

func finishReason(stopReason string) forge.FinishReason {
	if stopReason == "tool_use" {
		return forge.FinishReasonToolUse
	}
	return forge.FinishReasonStop
}
