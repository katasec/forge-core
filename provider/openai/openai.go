// Package openai implements forge.Provider using the OpenAI Responses API.
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/katasec/forge-core"
)

// Provider implements forge.Provider using the OpenAI Responses API.
type Provider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// New creates an OpenAI provider using the Responses API.
func New(apiKey string, model Model, opts ...Option) *Provider {
	p := &Provider{
		baseURL: "https://api.openai.com/v1",
		apiKey:  apiKey,
		model:   string(model),
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Capabilities describes the OpenAI provider features Forge currently supports.
func (p *Provider) Capabilities() forge.Capabilities {
	return forge.Capabilities{
		Images:     true,
		Usage:      true,
		Production: true,
	}
}

// --- OpenAI Responses API request/response types ---

type request struct {
	Model        string      `json:"model"`
	Input        []inputItem `json:"input"`
	Instructions string      `json:"instructions,omitempty"`
}

type inputItem struct {
	Role    string         `json:"role"`
	Content []contentInput `json:"content"`
}

type contentInput struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type response struct {
	Output []outputItem `json:"output"`
	Usage  usage        `json:"usage"`
}

type outputItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	Content []contentOutput `json:"content,omitempty"`
}

type contentOutput struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type usage struct {
	InputTokens         int                 `json:"input_tokens"`
	InputTokensDetails  inputTokensDetails  `json:"input_tokens_details"`
	OutputTokens        int                 `json:"output_tokens"`
	OutputTokensDetails outputTokensDetails `json:"output_tokens_details"`
	TotalTokens         int                 `json:"total_tokens"`
}

type inputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type outputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// Generate sends a request to the OpenAI Responses API.
func (p *Provider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
	input, err := convertMessages(req.Messages)
	if err != nil {
		return nil, err
	}

	body := request{
		Model:        p.model,
		Input:        input,
		Instructions: req.SystemPrompt,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/responses", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

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
		switch block.Type {
		case forge.ContentTypeText:
			contentType := "input_text"
			if role == forge.RoleAssistant {
				contentType = "output_text"
			}
			content = append(content, contentInput{Type: contentType, Text: block.Text})
		case forge.ContentTypeImage:
			if role != forge.RoleUser {
				return nil, fmt.Errorf("openai image content is only supported for user messages")
			}
			if block.Image == nil {
				return nil, fmt.Errorf("image content block missing image data")
			}
			imageURL, err := openAIImageURL(*block.Image)
			if err != nil {
				return nil, err
			}
			content = append(content, contentInput{Type: "input_image", ImageURL: imageURL})
		case forge.ContentTypeToolCall, forge.ContentTypeToolResult:
			return nil, fmt.Errorf("openai provider does not support tool content yet")
		default:
			return nil, fmt.Errorf("unsupported content block type: %s", block.Type)
		}
	}
	return content, nil
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

func convertResponse(apiResp response) []forge.Message {
	var messages []forge.Message
	for _, item := range apiResp.Output {
		if item.Type != "message" {
			continue
		}

		var blocks []forge.ContentBlock
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				blocks = append(blocks, forge.Text(content.Text))
			}
		}
		if len(blocks) == 0 {
			continue
		}

		role := forge.RoleAssistant
		if item.Role != "" {
			role = forge.Role(item.Role)
		}
		messages = append(messages, forge.Message{Role: role, Content: blocks})
	}
	return messages
}
