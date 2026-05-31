// Package anthropic implements forge.Provider using the Anthropic Messages API.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/katasec/forge-core"
)

// Provider implements forge.Provider using the Anthropic Messages API.
type Provider struct {
	apiKey string
	model  string
	client *http.Client
}

// New creates an Anthropic provider for the given API key and model.
func New(apiKey, model string) *Provider {
	return &Provider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Capabilities describes the Anthropic provider features Forge currently supports.
func (p *Provider) Capabilities() forge.Capabilities {
	return forge.Capabilities{
		Usage:      true,
		Production: true,
	}
}

// --- Anthropic API request/response types ---

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type response struct {
	Content    []content `json:"content"`
	StopReason string    `json:"stop_reason"`
	Usage      usage     `json:"usage"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Generate sends a request to the Anthropic Messages API.
func (p *Provider) Generate(ctx context.Context, req forge.ProviderRequest) (*forge.ProviderResponse, error) {
	// Convert forge messages to Anthropic format.
	var msgs []message
	for _, m := range req.Messages {
		if m.Role == forge.RoleSystem {
			continue // system prompt handled separately
		}
		msgs = append(msgs, message{
			Role:    string(m.Role),
			Content: m.Text(),
		})
	}

	body := request{
		Model:     p.model,
		MaxTokens: 1024,
		System:    req.SystemPrompt,
		Messages:  msgs,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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
		return nil, fmt.Errorf("anthropic API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Extract text content.
	var textContent string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			textContent = c.Text
			break
		}
	}

	finishReason := forge.FinishReasonStop
	if apiResp.StopReason == "tool_use" {
		finishReason = forge.FinishReasonToolUse
	}

	return &forge.ProviderResponse{
		Messages:     []forge.Message{forge.AssistantText(textContent)},
		FinishReason: finishReason,
		Usage: forge.TokenUsage{
			InputTokens:  apiResp.Usage.InputTokens,
			OutputTokens: apiResp.Usage.OutputTokens,
		},
	}, nil
}
