package anthropic

import (
	"encoding/json"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"github.com/katasec/kiln"
	"github.com/katasec/kiln/message"
	"github.com/katasec/kiln/tool"
)

// buildRequest adapts a Forge provider request into Anthropic Messages parameters.
func (p *AnthropicProvider) buildRequest(req kiln.ProviderRequest) anthropicsdk.MessageNewParams {
	return anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.Model(p.model),
		MaxTokens: int64(p.maxTokens),
		System:    systemPrompt(req.SystemPrompt),
		Messages:  toAnthropicMessages(req.Messages),
		Tools:     toAnthropicTools(req.Tools),
	}
}

// systemPrompt returns Anthropic's top-level system prompt blocks.
func systemPrompt(prompt string) []anthropicsdk.TextBlockParam {
	if prompt == "" {
		return nil
	}
	return []anthropicsdk.TextBlockParam{{Text: prompt}}
}

// toAnthropicTools converts Forge tool definitions into Anthropic tool params.
func toAnthropicTools(defs []kiln.ToolDefinition) []anthropicsdk.ToolUnionParam {
	if len(defs) == 0 {
		return nil
	}

	tools := make([]anthropicsdk.ToolUnionParam, 0, len(defs))
	for _, d := range defs {
		schema := tool.ObjectSchema(d.Schema.Parameters)
		param := anthropicsdk.ToolUnionParamOfTool(anthropicsdk.ToolInputSchemaParam{
			Properties: tool.SchemaProperties(schema),
			Required:   tool.SchemaRequired(schema),
		}, d.Name)
		if d.Description != "" && param.OfTool != nil {
			param.OfTool.Description = anthropicsdk.String(d.Description)
		}
		tools = append(tools, param)
	}
	return tools
}

// toAnthropicMessages converts Forge conversation messages into Anthropic message params.
// Messages that carry no renderable content are dropped: Anthropic rejects empty blocks.
func toAnthropicMessages(messages []kiln.Message) []anthropicsdk.MessageParam {
	out := make([]anthropicsdk.MessageParam, 0, len(messages))
	for _, m := range messages {
		if m.Role == kiln.RoleSystem {
			continue
		}
		converted, ok := toAnthropicMessage(m)
		if !ok {
			continue
		}
		out = append(out, converted)
	}
	return out
}

// toAnthropicMessage converts one Forge message into an Anthropic message param.
// Tool results travel as user messages, which is the shape Anthropic expects.
func toAnthropicMessage(msg kiln.Message) (anthropicsdk.MessageParam, bool) {
	switch msg.Role {
	case kiln.RoleAssistant:
		blocks := assistantBlocks(msg)
		if len(blocks) == 0 {
			return anthropicsdk.MessageParam{}, false
		}
		return anthropicsdk.NewAssistantMessage(blocks...), true
	case kiln.RoleTool:
		blocks := toolResultBlocks(msg)
		if len(blocks) == 0 {
			return anthropicsdk.MessageParam{}, false
		}
		return anthropicsdk.NewUserMessage(blocks...), true
	default:
		text := msg.Text()
		if text == "" {
			return anthropicsdk.MessageParam{}, false
		}
		return anthropicsdk.NewUserMessage(anthropicsdk.NewTextBlock(text)), true
	}
}

// assistantBlocks renders assistant text and tool calls as Anthropic content blocks.
func assistantBlocks(msg kiln.Message) []anthropicsdk.ContentBlockParamUnion {
	var blocks []anthropicsdk.ContentBlockParamUnion
	if text := msg.Text(); text != "" {
		blocks = append(blocks, anthropicsdk.NewTextBlock(text))
	}
	for _, call := range msg.ToolCalls() {
		blocks = append(blocks, anthropicsdk.NewToolUseBlock(call.ID, call.Arguments, call.Name))
	}
	return blocks
}

// toolResultBlocks renders tool results as Anthropic tool_result blocks.
func toolResultBlocks(msg kiln.Message) []anthropicsdk.ContentBlockParamUnion {
	results := msg.ToolResults()
	blocks := make([]anthropicsdk.ContentBlockParamUnion, 0, len(results))
	for _, r := range results {
		blocks = append(blocks, anthropicsdk.NewToolResultBlock(r.CallID, r.Content, r.IsError))
	}
	return blocks
}

// providerResponse adapts an Anthropic message response into Forge's provider response.
func providerResponse(apiResp *anthropicsdk.Message) *kiln.ProviderResponse {
	return &kiln.ProviderResponse{
		Messages:     []kiln.Message{{Role: kiln.RoleAssistant, Content: fromAnthropicContent(apiResp.Content)}},
		FinishReason: finishReason(apiResp.StopReason),
		Usage: kiln.TokenUsage{
			InputTokens:  int(apiResp.Usage.InputTokens),
			OutputTokens: int(apiResp.Usage.OutputTokens),
		},
	}
}

// fromAnthropicContent converts Anthropic response blocks into Forge content blocks.
func fromAnthropicContent(content []anthropicsdk.ContentBlockUnion) []kiln.ContentBlock {
	blocks := make([]kiln.ContentBlock, 0, len(content))
	for _, c := range content {
		switch c.Type {
		case "text":
			if c.Text != "" {
				blocks = append(blocks, message.Text(c.Text))
			}
		case "tool_use":
			blocks = append(blocks, message.ToolCall(kiln.ToolCall{
				ID:        c.ID,
				Name:      c.Name,
				Arguments: json.RawMessage(c.Input),
			}))
		}
	}
	return blocks
}

// finishReason maps Anthropic stop reasons onto Forge finish reasons.
func finishReason(stopReason anthropicsdk.StopReason) kiln.FinishReason {
	if stopReason == anthropicsdk.StopReasonToolUse {
		return kiln.FinishReasonToolUse
	}
	return kiln.FinishReasonStop
}
