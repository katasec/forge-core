package anthropic

import (
	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"github.com/katasec/forge-core"
	"github.com/katasec/forge-core/message"
)

// buildRequest adapts a Forge provider request into Anthropic Messages parameters.
func (p *AnthropicProvider) buildRequest(req forge.ProviderRequest) anthropicsdk.MessageNewParams {
	return anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(p.model),
		MaxTokens: 1024,
		System:    systemPrompt(req.SystemPrompt),
		Messages:  toAnthropicMessages(req.Messages),
	}
}

// systemPrompt returns Anthropic's top-level system prompt blocks.
func systemPrompt(prompt string) []anthropicsdk.TextBlockParam {
	if prompt == "" {
		return nil
	}
	return []anthropicsdk.TextBlockParam{{Text: prompt}}
}

// toAnthropicMessages converts Forge conversation messages into Anthropic message params.
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

// toAnthropicMessage converts one Forge message into an Anthropic message param.
func toAnthropicMessage(msg forge.Message) anthropicsdk.MessageParam {
	if msg.Role == forge.RoleAssistant {
		return anthropicsdk.NewAssistantMessage(anthropicsdk.NewTextBlock(msg.Text()))
	}
	return anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(msg.Text()))
}

// providerResponse adapts an Anthropic message response into Forge's provider response.
func providerResponse(apiResp *anthropicsdk.Message) *forge.ProviderResponse {
	return &forge.ProviderResponse{
		Messages:     []forge.Message{message.AssistantText(textFromAnthropic(apiResp.Content))},
		FinishReason: finishReason(apiResp.StopReason),
		Usage: forge.TokenUsage{
			InputTokens:  int(apiResp.Usage.InputTokens),
			OutputTokens: int(apiResp.Usage.OutputTokens),
		},
	}
}

// textFromAnthropic returns the first text block from an Anthropic response.
func textFromAnthropic(content []anthropicsdk.ContentBlockUnion) string {
	for _, c := range content {
		if c.Type == "text" {
			return c.Text
		}
	}
	return ""
}

// finishReason maps Anthropic stop reasons onto Forge finish reasons.
func finishReason(stopReason anthropicsdk.StopReason) forge.FinishReason {
	if stopReason == anthropicsdk.StopReasonToolUse {
		return forge.FinishReasonToolUse
	}
	return forge.FinishReasonStop
}
