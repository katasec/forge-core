package xai

import (
	"encoding/json"

	"github.com/katasec/forge-core"
	"github.com/katasec/forge-core/message"
)

func convertMessages(msgs []forge.Message, systemPrompt string) []inputItem {
	var items []inputItem

	if systemPrompt != "" {
		items = append(items, inputItem{Role: "system", Content: systemPrompt})
	}

	for _, m := range msgs {
		if m.Role == forge.RoleSystem {
			continue
		}
		items = append(items, convertMessage(m)...)
	}

	return items
}

func convertMessage(m forge.Message) []inputItem {
	if m.Role == forge.RoleTool && len(m.ToolResults()) > 0 {
		return convertToolResults(m.ToolResults())
	}
	return []inputItem{{
		Role:    string(m.Role),
		Content: m.Text(),
	}}
}

func convertToolResults(results []forge.ToolResult) []inputItem {
	items := make([]inputItem, 0, len(results))
	for _, tr := range results {
		items = append(items, inputItem{
			Type:   "function_call_output",
			CallID: tr.CallID,
			Output: tr.Content,
		})
	}
	return items
}

func convertTools(defs []forge.ToolDefinition) []requestTool {
	tools := make([]requestTool, 0, len(defs))
	for _, d := range defs {
		tools = append(tools, requestTool{
			Type:        "function",
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.Schema.Parameters,
		})
	}
	return tools
}

func providerResponse(resp *response) (*forge.ProviderResponse, []Citation) {
	content, toolCalls, citations := convertOutput(resp.Output)

	blocks := []forge.ContentBlock{}
	if content != "" {
		blocks = append(blocks, forge.Text(content))
	}
	for _, call := range toolCalls {
		blocks = append(blocks, message.ToolCall(call))
	}

	return &forge.ProviderResponse{
		Messages:     []forge.Message{{Role: forge.RoleAssistant, Content: blocks}},
		FinishReason: finishReason(toolCalls),
		Usage: forge.TokenUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}, citations
}

func convertOutput(output []outputItem) (string, []forge.ToolCall, []Citation) {
	var content string
	var toolCalls []forge.ToolCall
	var citations []Citation

	for _, item := range output {
		switch item.Type {
		case "function_call":
			toolCalls = append(toolCalls, forge.ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: json.RawMessage(item.Arguments),
			})
		case "message":
			text, found := convertMessageOutput(item.Content)
			content += text
			citations = append(citations, found...)
		}
	}

	return content, toolCalls, citations
}

func convertMessageOutput(content []contentItem) (string, []Citation) {
	var text string
	var citations []Citation
	for _, c := range content {
		if c.Type != "output_text" {
			continue
		}
		text += c.Text
		citations = append(citations, convertAnnotations(c.Annotations)...)
	}
	return text, citations
}

func convertAnnotations(annotations []annotation) []Citation {
	var citations []Citation
	for _, a := range annotations {
		if a.Type == "url_citation" {
			citations = append(citations, Citation{
				URL:        a.URL,
				Title:      a.Title,
				Source:     "web",
				StartIndex: a.StartIndex,
				EndIndex:   a.EndIndex,
			})
		}
	}
	return citations
}

func finishReason(toolCalls []forge.ToolCall) forge.FinishReason {
	if len(toolCalls) > 0 {
		return forge.FinishReasonToolUse
	}
	return forge.FinishReasonStop
}
