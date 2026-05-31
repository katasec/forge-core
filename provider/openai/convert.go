package openai

import (
	"encoding/base64"
	"fmt"

	"github.com/katasec/forge-core"
)

func (p *Provider) buildRequest(req forge.ProviderRequest) (request, error) {
	input, err := convertMessages(req.Messages)
	if err != nil {
		return request{}, err
	}
	return request{
		Model:        p.model,
		Input:        input,
		Instructions: req.SystemPrompt,
	}, nil
}

func providerResponse(apiResp *response) (*forge.ProviderResponse, error) {
	messages := convertResponse(apiResp)
	if len(messages) == 0 {
		return nil, fmt.Errorf("no assistant messages in response")
	}

	return &forge.ProviderResponse{
		Messages:     messages,
		FinishReason: forge.FinishReasonStop,
		Usage: forge.TokenUsage{
			InputTokens:           apiResp.Usage.InputTokens,
			CachedInputTokens:     apiResp.Usage.InputTokensDetails.CachedTokens,
			OutputTokens:          apiResp.Usage.OutputTokens,
			ReasoningOutputTokens: apiResp.Usage.OutputTokensDetails.ReasoningTokens,
			TotalTokens:           apiResp.Usage.TotalTokens,
		},
	}, nil
}

func convertMessages(messages []forge.Message) ([]inputItem, error) {
	items := make([]inputItem, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == forge.RoleSystem {
			continue
		}

		content, err := convertContent(msg.Role, msg.Content)
		if err != nil {
			return nil, err
		}
		if len(content) == 0 {
			continue
		}

		items = append(items, inputItem{
			Role:    string(msg.Role),
			Content: content,
		})
	}
	return items, nil
}

func convertContent(role forge.Role, blocks []forge.ContentBlock) ([]contentInput, error) {
	content := make([]contentInput, 0, len(blocks))
	for _, block := range blocks {
		converted, err := convertContentBlock(role, block)
		if err != nil {
			return nil, err
		}
		content = append(content, converted)
	}
	return content, nil
}

func convertContentBlock(role forge.Role, block forge.ContentBlock) (contentInput, error) {
	switch block.Type {
	case forge.ContentTypeText:
		return textContent(role, block.Text), nil
	case forge.ContentTypeImage:
		return imageContent(role, block)
	case forge.ContentTypeToolCall, forge.ContentTypeToolResult:
		return contentInput{}, fmt.Errorf("openai provider does not support tool content yet")
	default:
		return contentInput{}, fmt.Errorf("unsupported content block type: %s", block.Type)
	}
}

func textContent(role forge.Role, text string) contentInput {
	contentType := "input_text"
	if role == forge.RoleAssistant {
		contentType = "output_text"
	}
	return contentInput{Type: contentType, Text: text}
}

func imageContent(role forge.Role, block forge.ContentBlock) (contentInput, error) {
	if role != forge.RoleUser {
		return contentInput{}, fmt.Errorf("openai image content is only supported for user messages")
	}
	if block.Image == nil {
		return contentInput{}, fmt.Errorf("image content block missing image data")
	}

	imageURL, err := openAIImageURL(*block.Image)
	if err != nil {
		return contentInput{}, err
	}
	return contentInput{Type: "input_image", ImageURL: imageURL}, nil
}

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

func convertResponse(apiResp *response) []forge.Message {
	var messages []forge.Message
	for _, item := range apiResp.Output {
		if item.Type != "message" {
			continue
		}

		msg, ok := convertOutputItem(item)
		if ok {
			messages = append(messages, msg)
		}
	}
	return messages
}

func convertOutputItem(item outputItem) (forge.Message, bool) {
	blocks := outputBlocks(item.Content)
	if len(blocks) == 0 {
		return forge.Message{}, false
	}

	role := forge.RoleAssistant
	if item.Role != "" {
		role = forge.Role(item.Role)
	}
	return forge.Message{Role: role, Content: blocks}, true
}

func outputBlocks(content []contentOutput) []forge.ContentBlock {
	var blocks []forge.ContentBlock
	for _, part := range content {
		if part.Type == "output_text" && part.Text != "" {
			blocks = append(blocks, forge.Text(part.Text))
		}
	}
	return blocks
}
