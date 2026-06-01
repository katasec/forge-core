package provider

import (
	"context"

	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/tool"
)

// FinishReason indicates why the agent loop terminated.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonToolUse   FinishReason = "tool_use"
	FinishReasonIterLimit FinishReason = "iter_limit"
	FinishReasonError     FinishReason = "error"
)

// TokenUsage tracks token consumption across provider calls.
type TokenUsage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens,omitempty"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens,omitempty"`
	TotalTokens           int `json:"total_tokens,omitempty"`
}

// Capabilities describes optional provider features.
type Capabilities struct {
	Tools      bool `json:"tools,omitempty"`
	Images     bool `json:"images,omitempty"`
	Streaming  bool `json:"streaming,omitempty"`
	Usage      bool `json:"usage,omitempty"`
	Local      bool `json:"local,omitempty"`
	Production bool `json:"production,omitempty"`
}

// CapabilityProvider is implemented by providers that can describe their features.
type CapabilityProvider interface {
	Capabilities() Capabilities
}

// Request is the input to a single LLM call.
type Request struct {
	Messages     []message.Message `json:"messages"`
	Tools        []tool.Definition `json:"tools,omitempty"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
}

// Response is the output of a single LLM call.
type Response struct {
	Messages     []message.Message `json:"messages"`
	FinishReason FinishReason      `json:"finish_reason"`
	Usage        TokenUsage        `json:"usage"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

// Provider makes a single LLM call. It does not loop.
type Provider interface {
	Generate(ctx context.Context, req Request) (*Response, error)
}
