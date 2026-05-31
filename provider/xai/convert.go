package xai

import (
	"encoding/json"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/katasec/forge-core"
	"github.com/katasec/forge-core/message"
)

// buildRequest adapts a Forge provider request into xAI's OpenAI-compatible parameters.
func (p *XAIProvider) buildRequest(req forge.ProviderRequest) (responses.ResponseNewParams, error) {
	input, err := toXAIMessages(req.Messages)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}

	apiReq := responses.ResponseNewParams{
		Model: shared.ResponsesModel(p.model),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
	}
	if req.SystemPrompt != "" {
		apiReq.Instructions = openaisdk.String(req.SystemPrompt)
	}
	return apiReq, nil
}

// toXAIMessages converts Forge conversation messages into xAI response input items.
func toXAIMessages(msgs []forge.Message) (responses.ResponseInputParam, error) {
	var items responses.ResponseInputParam

	for _, m := range msgs {
		if m.Role == forge.RoleSystem {
			continue
		}
		converted, err := toXAIMessage(m)
		if err != nil {
			return nil, err
		}
		items = append(items, converted...)
	}
	return items, nil
}

// toXAIMessage converts one Forge message into xAI response input items.
func toXAIMessage(m forge.Message) ([]responses.ResponseInputItemUnionParam, error) {
	if m.Role == forge.RoleTool && len(m.ToolResults()) > 0 {
		return toXAIToolResults(m.ToolResults()), nil
	}
	if len(m.ToolCalls()) > 0 {
		return nil, nil
	}

	role := responses.EasyInputMessageRole(m.Role)
	item := responses.ResponseInputItemParamOfMessage(m.Text(), role)
	return []responses.ResponseInputItemUnionParam{item}, nil
}

// toXAIToolResults converts Forge tool results into xAI function call outputs.
func toXAIToolResults(results []forge.ToolResult) []responses.ResponseInputItemUnionParam {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(results))
	for _, tr := range results {
		items = append(items, responses.ResponseInputItemParamOfFunctionCallOutput(tr.CallID, tr.Content))
	}
	return items
}

// toXAITools converts Forge tool definitions into xAI request tools.
func toXAITools(defs []forge.ToolDefinition) []requestTool {
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

// providerResponse adapts an xAI response into Forge's provider response and citations.
func providerResponse(resp *response) (*forge.ProviderResponse, []Citation) {
	content, toolCalls, citations := fromXAIOutput(resp.Output)

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

// fromXAIOutput extracts text, function calls, and citations from xAI output items.
func fromXAIOutput(output []outputItem) (string, []forge.ToolCall, []Citation) {
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
			text, found := fromXAIMessageOutput(item.Content)
			content += text
			citations = append(citations, found...)
		}
	}

	return content, toolCalls, citations
}

// fromXAIMessageOutput extracts text and citations from xAI message content.
func fromXAIMessageOutput(content []contentItem) (string, []Citation) {
	var text string
	var citations []Citation
	for _, c := range content {
		if c.Type != "output_text" {
			continue
		}
		text += c.Text
		citations = append(citations, fromXAIAnnotations(c.Annotations)...)
	}
	return text, citations
}

// fromXAIAnnotations converts xAI URL annotations into Forge citations.
func fromXAIAnnotations(annotations []annotation) []Citation {
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

// finishReason reports tool_use when the xAI response contains function calls.
func finishReason(toolCalls []forge.ToolCall) forge.FinishReason {
	if len(toolCalls) > 0 {
		return forge.FinishReasonToolUse
	}
	return forge.FinishReasonStop
}
