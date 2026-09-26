package kiln

import (
	"github.com/katasec/kiln/executor"
	"github.com/katasec/kiln/executor/sequential"
	"github.com/katasec/kiln/internal/runtime"
	"github.com/katasec/kiln/memory"
	"github.com/katasec/kiln/memory/inmem"
	"github.com/katasec/kiln/message"
	"github.com/katasec/kiln/middleware"
	"github.com/katasec/kiln/provider"
	"github.com/katasec/kiln/skill"
	"github.com/katasec/kiln/skill/markdown"
	"github.com/katasec/kiln/tool"
	"github.com/katasec/kiln/tool/registry"
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

type SkillKind = skill.Kind
type SkillSpec = skill.Spec
type SkillInput = skill.Input
type SkillResult = skill.Result
type SkillStatus = skill.Status
type SkillRunner = skill.Runner
type SkillRegistry = skill.Registry
type SkillManifest = skill.Manifest
type SkillOutcome = skill.Outcome
type MarkdownRunner = markdown.Runner

const (
	SkillKindContext = skill.KindContext
	SkillKindProcess = skill.KindProcess

	SkillStatusSuccess = skill.StatusSuccess
	SkillStatusError   = skill.StatusError

	SkillOutcomeExecute = skill.OutcomeExecute
	SkillOutcomeExpose  = skill.OutcomeExpose
	SkillOutcomeRefuse  = skill.OutcomeRefuse
)
