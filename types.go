package forge

import (
	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/provider"
	"github.com/katasec/forge-core/tool"
)

type Role = message.Role

const (
	RoleUser      = message.RoleUser
	RoleAssistant = message.RoleAssistant
	RoleTool      = message.RoleTool
	RoleSystem    = message.RoleSystem
)

type Message = message.Message
type ContentBlock = message.ContentBlock
type ContentType = message.ContentType
type ImageContent = message.ImageContent

const (
	ContentTypeText       = message.ContentTypeText
	ContentTypeImage      = message.ContentTypeImage
	ContentTypeToolCall   = message.ContentTypeToolCall
	ContentTypeToolResult = message.ContentTypeToolResult
)

func Text(content string) ContentBlock {
	return message.Text(content)
}

func ImageURL(url string) ContentBlock {
	return message.ImageURL(url)
}

func ImageBytes(data []byte, mediaType string) ContentBlock {
	return message.ImageBytes(data, mediaType)
}

func ToolCallBlock(call ToolCall) ContentBlock {
	return message.ToolCall(call)
}

func ToolResultBlock(result ToolResult) ContentBlock {
	return message.ToolResult(result)
}

func UserMessage(blocks ...ContentBlock) Message {
	return message.UserMessage(blocks...)
}

func UserText(content string) Message {
	return message.UserText(content)
}

func AssistantText(content string) Message {
	return message.AssistantText(content)
}

type ToolCall = tool.Call
type ToolResult = tool.Result
type ToolError = tool.Error

type FinishReason = provider.FinishReason

const (
	FinishReasonStop      = provider.FinishReasonStop
	FinishReasonToolUse   = provider.FinishReasonToolUse
	FinishReasonIterLimit = provider.FinishReasonIterLimit
	FinishReasonError     = provider.FinishReasonError
)

type TokenUsage = provider.TokenUsage
type Capabilities = provider.Capabilities
type CapabilityProvider = provider.CapabilityProvider

type ErrorPolicy string

const (
	ErrorPolicyStop     ErrorPolicy = "stop"
	ErrorPolicyContinue ErrorPolicy = "continue"
)
