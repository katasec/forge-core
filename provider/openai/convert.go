package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/katasec/kiln"
	"github.com/katasec/kiln/message"
	"github.com/katasec/kiln/tool"
)

// buildRequest adapts a Forge provider request into OpenAI Responses parameters.
func (p *OpenAIProvider) buildRequest(req kiln.ProviderRequest) (responses.ResponseNewParams, error) {
	input, err := toOpenAIMessages(req.Messages)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}
	return responses.ResponseNewParams{
		Model: shared.ResponsesModel(p.model),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
		Instructions: openaisdk.String(req.SystemPrompt),
		Tools:        toOpenAITools(req.Tools),
	}, nil
}

// toOpenAITools converts Forge tool definitions into OpenAI function tools.
// Strict mode is left off: it imposes schema rules the reflected schemas do
// not necessarily satisfy.
func toOpenAITools(defs []kiln.ToolDefinition) []responses.ToolUnionParam {
	if len(defs) == 0 {
		return nil
	}

	tools := make([]responses.ToolUnionParam, 0, len(defs))
	for _, d := range defs {
		param := responses.ToolParamOfFunction(d.Name, tool.ObjectSchema(d.Schema.Parameters), false)
		if d.Description != "" && param.OfFunction != nil {
			param.OfFunction.Description = openaisdk.String(d.Description)
		}
		tools = append(tools, param)
	}
	return tools
}

// providerResponse adapts an OpenAI Responses result into Forge's provider response.
func providerResponse(apiResp *responses.Response) (*kiln.ProviderResponse, error) {
	blocks := fromOpenAIOutput(apiResp)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no assistant messages in response")
	}

	return &kiln.ProviderResponse{
		Messages:     []kiln.Message{{Role: kiln.RoleAssistant, Content: blocks}},
		FinishReason: finishReason(blocks),
		Usage: kiln.TokenUsage{
			InputTokens:           int(apiResp.Usage.InputTokens),
			CachedInputTokens:     int(apiResp.Usage.InputTokensDetails.CachedTokens),
			OutputTokens:          int(apiResp.Usage.OutputTokens),
			ReasoningOutputTokens: int(apiResp.Usage.OutputTokensDetails.ReasoningTokens),
			TotalTokens:           int(apiResp.Usage.TotalTokens),
		},
	}, nil
}

// toOpenAIMessages converts Forge conversation messages into OpenAI response input items.
func toOpenAIMessages(messages []kiln.Message) (responses.ResponseInputParam, error) {
	items := make(responses.ResponseInputParam, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == kiln.RoleSystem {
			continue
		}

		converted, err := toOpenAIMessage(msg)
		if err != nil {
			return nil, err
		}
		items = append(items, converted...)
	}
	return items, nil
}

// toOpenAIMessage converts one Forge message into OpenAI response input items.
// Tool calls and tool results are separate items in the Responses API, so a
// single Forge message can expand into several.
func toOpenAIMessage(msg kiln.Message) ([]responses.ResponseInputItemUnionParam, error) {
	if results := msg.ToolResults(); len(results) > 0 {
		return toOpenAIToolResults(results), nil
	}
	if calls := msg.ToolCalls(); len(calls) > 0 {
		return toOpenAIToolCalls(msg, calls), nil
	}

	// An assistant turn replayed from memory must carry output_text, not the
	// input_text parts a user turn uses: OpenAI rejects input_text on an
	// assistant message. The plain-string form lets the API pick the right one.
	if msg.Role == kiln.RoleAssistant {
		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(msg.Text(), responses.EasyInputMessageRole(msg.Role)),
		}, nil
	}

	content, err := toOpenAIContent(msg.Role, msg.Content)
	if err != nil {
		return nil, err
	}
	return []responses.ResponseInputItemUnionParam{
		responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRole(msg.Role)),
	}, nil
}

// toOpenAIToolCalls renders assistant text plus its function calls as items.
func toOpenAIToolCalls(msg kiln.Message, calls []kiln.ToolCall) []responses.ResponseInputItemUnionParam {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(calls)+1)
	if text := msg.Text(); text != "" {
		items = append(items, responses.ResponseInputItemParamOfMessage(text, responses.EasyInputMessageRole(msg.Role)))
	}
	for _, call := range calls {
		items = append(items, responses.ResponseInputItemParamOfFunctionCall(string(call.Arguments), call.ID, call.Name))
	}
	return items
}

// toOpenAIToolResults renders tool results as function call output items.
func toOpenAIToolResults(results []kiln.ToolResult) []responses.ResponseInputItemUnionParam {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(results))
	for _, r := range results {
		item := responses.ResponseInputItemParamOfFunctionCallOutput(r.Content)
		item.OfFunctionCallOutput.CallID = param.NewOpt(r.CallID)
		items = append(items, item)
	}
	return items
}

// fromOpenAIOutput converts OpenAI output items into Forge content blocks.
func fromOpenAIOutput(apiResp *responses.Response) []kiln.ContentBlock {
	var blocks []kiln.ContentBlock
	if text := apiResp.OutputText(); text != "" {
		blocks = append(blocks, message.Text(text))
	}
	for _, item := range apiResp.Output {
		if item.Type != "function_call" {
			continue
		}
		blocks = append(blocks, message.ToolCall(kiln.ToolCall{
			ID:        item.CallID,
			Name:      item.Name,
			Arguments: json.RawMessage(item.Arguments.OfString),
		}))
	}
	return blocks
}

// finishReason reports tool use when the model asked for at least one call.
func finishReason(blocks []kiln.ContentBlock) kiln.FinishReason {
	for _, b := range blocks {
		if b.Type == kiln.ContentTypeToolCall {
			return kiln.FinishReasonToolUse
		}
	}
	return kiln.FinishReasonStop
}

// toOpenAIContent converts Forge content blocks into OpenAI message content parts.
func toOpenAIContent(role kiln.Role, blocks []kiln.ContentBlock) (responses.ResponseInputMessageContentListParam, error) {
	content := make(responses.ResponseInputMessageContentListParam, 0, len(blocks))
	for _, block := range blocks {
		converted, err := toOpenAIContentBlock(role, block)
		if err != nil {
			return nil, err
		}
		content = append(content, converted)
	}
	return content, nil
}

// toOpenAIContentBlock converts one Forge content block into an OpenAI content part.
func toOpenAIContentBlock(role kiln.Role, block kiln.ContentBlock) (responses.ResponseInputContentUnionParam, error) {
	switch block.Type {
	case kiln.ContentTypeText:
		return toOpenAITextContent(role, block.Text), nil
	case kiln.ContentTypeImage:
		return toOpenAIImageContent(role, block)
	case kiln.ContentTypeToolCall, kiln.ContentTypeToolResult:
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("tool content must be converted as its own input item, not as message content")
	default:
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("unsupported content block type: %s", block.Type)
	}
}

// toOpenAITextContent wraps text as an OpenAI input text content part.
func toOpenAITextContent(_ kiln.Role, text string) responses.ResponseInputContentUnionParam {
	return responses.ResponseInputContentParamOfInputText(text)
}

// toOpenAIImageContent wraps Forge image content as an OpenAI input image content part.
func toOpenAIImageContent(role kiln.Role, block kiln.ContentBlock) (responses.ResponseInputContentUnionParam, error) {
	if role != kiln.RoleUser {
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("openai image content is only supported for user messages")
	}
	if block.Image == nil {
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("image content block missing image data")
	}

	imageURL, err := openAIImageURL(*block.Image)
	if err != nil {
		return responses.ResponseInputContentUnionParam{}, err
	}
	return responses.ResponseInputContentUnionParam{
		OfInputImage: &responses.ResponseInputImageParam{
			Detail:   responses.ResponseInputImageDetailAuto,
			ImageURL: openaisdk.String(imageURL),
		},
	}, nil
}

// openAIImageURL returns the URL or data URL OpenAI expects for image input.
func openAIImageURL(image kiln.ImageContent) (string, error) {
	if image.URL != "" {
		return image.URL, nil
	}
	if len(image.Data) == 0 {
		return "", fmt.Errorf("image content requires URL or data")
	}
	if image.MediaType == "" {
		return "", fmt.Errorf("image bytes require media type")
	}
	encoded := base64.StdEncoding.EncodeToString(image.Data)
	return fmt.Sprintf("data:%s;base64,%s", image.MediaType, encoded), nil
}
