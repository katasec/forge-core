package forge

import (
	"context"

	"github.com/katasec/forge-core/executor"
	"github.com/katasec/forge-core/executor/sequential"
	"github.com/katasec/forge-core/internal/runtime"
	"github.com/katasec/forge-core/memory"
	"github.com/katasec/forge-core/memory/inmem"
	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/middleware"
	"github.com/katasec/forge-core/provider"
	"github.com/katasec/forge-core/tool"
	"github.com/katasec/forge-core/tool/registry"
)

type AgentRequest = runtime.AgentRequest
type AgentResponse = runtime.AgentResponse
type Config = runtime.Config

type ProviderRequest = provider.Request
type ProviderResponse = provider.Response
type Provider = provider.Provider
type Capabilities = provider.Capabilities
type CapabilityProvider = provider.CapabilityProvider
type TokenUsage = provider.TokenUsage
type FinishReason = provider.FinishReason

const (
	FinishReasonStop      = provider.FinishReasonStop
	FinishReasonToolUse   = provider.FinishReasonToolUse
	FinishReasonIterLimit = provider.FinishReasonIterLimit
	FinishReasonError     = provider.FinishReasonError
)

type Message = message.Message
type Role = message.Role
type ContentBlock = message.ContentBlock
type ContentType = message.ContentType
type ImageContent = message.ImageContent

const (
	RoleUser      = message.RoleUser
	RoleAssistant = message.RoleAssistant
	RoleTool      = message.RoleTool
	RoleSystem    = message.RoleSystem
)

const (
	ContentTypeText       = message.ContentTypeText
	ContentTypeImage      = message.ContentTypeImage
	ContentTypeToolCall   = message.ContentTypeToolCall
	ContentTypeToolResult = message.ContentTypeToolResult
)

type Tool = tool.Tool
type ToolSchema = tool.Schema
type ToolDefinition = tool.Definition
type ToolCall = tool.Call
type ToolResult = tool.Result
type ToolError = tool.Error
type ToolRegistry = registry.Registry
type ToolExecutor = executor.Executor
type SequentialExecutor = sequential.Executor

type MemoryStore = memory.Store
type InMemoryStore = inmem.Store

type RunFunc = middleware.RunFunc
type Middleware = middleware.Middleware

type ErrorPolicy = runtime.ErrorPolicy

const (
	ErrorPolicyStop     = runtime.ErrorPolicyStop
	ErrorPolicyContinue = runtime.ErrorPolicyContinue
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

func Func[In any, Out any](name, description string, fn func(ctx context.Context, input In) (Out, error)) Tool {
	return tool.Func(name, description, fn)
}

func NewToolRegistry() *ToolRegistry {
	return registry.New()
}

func NewInMemoryStore() *InMemoryStore {
	return inmem.New()
}
