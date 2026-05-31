package openai

import (
	"encoding/base64"
	"fmt"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/katasec/forge-core"
)

// buildRequest adapts a Forge provider request into OpenAI Responses parameters.
func (p *OpenAIProvider) buildRequest(req forge.ProviderRequest) (responses.ResponseNewParams, error) {
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
	}, nil
}

// providerResponse adapts an OpenAI Responses result into Forge's provider response.
func providerResponse(apiResp *responses.Response) (*forge.ProviderResponse, error) {
	text := apiResp.OutputText()
	if text == "" {
		return nil, fmt.Errorf("no assistant messages in response")
	}

	return &forge.ProviderResponse{
		Messages:     []forge.Message{forge.AssistantText(text)},
		FinishReason: forge.FinishReasonStop,
		Usage: forge.TokenUsage{
			InputTokens:           int(apiResp.Usage.InputTokens),
			CachedInputTokens:     int(apiResp.Usage.InputTokensDetails.CachedTokens),
			OutputTokens:          int(apiResp.Usage.OutputTokens),
			ReasoningOutputTokens: int(apiResp.Usage.OutputTokensDetails.ReasoningTokens),
			TotalTokens:           int(apiResp.Usage.TotalTokens),
		},
	}, nil
}

// toOpenAIMessages converts Forge conversation messages into OpenAI response input items.
func toOpenAIMessages(messages []forge.Message) (responses.ResponseInputParam, error) {
	items := make(responses.ResponseInputParam, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == forge.RoleSystem {
			continue
		}

		item, err := toOpenAIMessage(msg)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// toOpenAIMessage converts one Forge message into one OpenAI response input item.
func toOpenAIMessage(msg forge.Message) (responses.ResponseInputItemUnionParam, error) {
	content, err := toOpenAIContent(msg.Role, msg.Content)
	if err != nil {
		return responses.ResponseInputItemUnionParam{}, err
	}
	return responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRole(msg.Role)), nil
}

// toOpenAIContent converts Forge content blocks into OpenAI message content parts.
func toOpenAIContent(role forge.Role, blocks []forge.ContentBlock) (responses.ResponseInputMessageContentListParam, error) {
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
func toOpenAIContentBlock(role forge.Role, block forge.ContentBlock) (responses.ResponseInputContentUnionParam, error) {
	switch block.Type {
	case forge.ContentTypeText:
		return toOpenAITextContent(role, block.Text), nil
	case forge.ContentTypeImage:
		return toOpenAIImageContent(role, block)
	case forge.ContentTypeToolCall, forge.ContentTypeToolResult:
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("openai provider does not support tool content yet")
	default:
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("unsupported content block type: %s", block.Type)
	}
}

// toOpenAITextContent wraps text as an OpenAI input text content part.
func toOpenAITextContent(_ forge.Role, text string) responses.ResponseInputContentUnionParam {
	return responses.ResponseInputContentParamOfInputText(text)
}

// toOpenAIImageContent wraps Forge image content as an OpenAI input image content part.
func toOpenAIImageContent(role forge.Role, block forge.ContentBlock) (responses.ResponseInputContentUnionParam, error) {
	if role != forge.RoleUser {
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
func openAIImageURL(image forge.ImageContent) (string, error) {
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
