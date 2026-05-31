package anthropic

import (
	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"github.com/katasec/forge-core"
)

func (p *AnthropicProvider) buildRequest(req forge.ProviderRequest) anthropicsdk.MessageNewParams {
	return anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(p.model),
		MaxTokens: 1024,
		System:    systemPrompt(req.SystemPrompt),
		Messages:  toAnthropicMessages(req.Messages),
	}
}

func systemPrompt(prompt string) []anthropicsdk.TextBlockParam {
	if prompt == "" {
		return nil
	}
	return []anthropicsdk.TextBlockParam{{Text: prompt}}
}

func toAnthropicMessages(messages []forge.Message) []anthropicsdk.MessageParam {
	out := make([]anthropicsdk.MessageParam, 0, len(messages))
	for _, m := range messages {
		if m.Role == forge.RoleSystem {
			continue
		}
		out = append(out, toAnthropicMessage(m))
	}
	return out
}

func toAnthropicMessage(message forge.Message) anthropicsdk.MessageParam {
	if message.Role == forge.RoleAssistant {
		return anthropicsdk.NewAssistantMessage(anthropicsdk.NewTextBlock(message.Text()))
	}
	return anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(message.Text()))
}

func providerResponse(apiResp *anthropicsdk.Message) *forge.ProviderResponse {
	return &forge.ProviderResponse{
		Messages:     []forge.Message{forge.AssistantText(textFromAnthropic(apiResp.Content))},
		FinishReason: finishReason(apiResp.StopReason),
		Usage: forge.TokenUsage{
			InputTokens:  int(apiResp.Usage.InputTokens),
			OutputTokens: int(apiResp.Usage.OutputTokens),
		},
	}
}

func textFromAnthropic(content []anthropicsdk.ContentBlockUnion) string {
	for _, c := range content {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}

func finishReason(stopReason anthropicsdk.StopReason) forge.FinishReason {
	if stopReason == anthropicsdk.StopReasonToolUse {
		return forge.FinishReasonToolUse
	}
	return forge.FinishReasonStop
}
