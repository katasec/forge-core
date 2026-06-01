package message

import (
	"strings"

	"github.com/katasec/forge-core/tool"
)

// Role identifies the sender of a message in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
)

// ContentType identifies the kind of content inside a message.
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolCall   ContentType = "tool_call"
	ContentTypeToolResult ContentType = "tool_result"
)

// ImageContent represents image input for multimodal providers.
type ImageContent struct {
	URL       string `json:"url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      []byte `json:"data,omitempty"`
}

// ContentBlock is one typed unit of message content.
type ContentBlock struct {
	Type       ContentType   `json:"type"`
	Text       string        `json:"text,omitempty"`
	Image      *ImageContent `json:"image,omitempty"`
	ToolCall   *tool.Call    `json:"tool_call,omitempty"`
	ToolResult *tool.Result  `json:"tool_result,omitempty"`
}

// Text creates a text content block.
func Text(content string) ContentBlock {
	return ContentBlock{Type: ContentTypeText, Text: content}
}

// ImageURL creates an image content block backed by a URL.
func ImageURL(url string) ContentBlock {
	return ContentBlock{Type: ContentTypeImage, Image: &ImageContent{URL: url}}
}

// ImageBytes creates an image content block backed by bytes.
func ImageBytes(data []byte, mediaType string) ContentBlock {
	return ContentBlock{Type: ContentTypeImage, Image: &ImageContent{Data: data, MediaType: mediaType}}
}

// ToolCall creates a tool-call content block.
func ToolCall(call tool.Call) ContentBlock {
	return ContentBlock{Type: ContentTypeToolCall, ToolCall: &call}
}

// ToolResult creates a tool-result content block.
func ToolResult(result tool.Result) ContentBlock {
	return ContentBlock{Type: ContentTypeToolResult, ToolResult: &result}
}

// Message represents a single message in a conversation.
type Message struct {
	ID      string         `json:"id,omitempty"`
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content,omitempty"`
}

// UserMessage creates a user-role message with the given content blocks.
func UserMessage(blocks ...ContentBlock) Message {
	return Message{Role: RoleUser, Content: blocks}
}

// UserText creates a user-role text message.
func UserText(content string) Message {
	return UserMessage(Text(content))
}

// AssistantText creates an assistant-role text message.
func AssistantText(content string) Message {
	return Message{Role: RoleAssistant, Content: []ContentBlock{Text(content)}}
}

// ToolMessage creates a tool-role message with tool results.
func ToolMessage(results ...tool.Result) Message {
	blocks := make([]ContentBlock, 0, len(results))
	for _, result := range results {
		blocks = append(blocks, ToolResult(result))
	}
	return Message{Role: RoleTool, Content: blocks}
}

// Text returns all text blocks joined together.
func (m Message) Text() string {
	var parts []string
	for _, block := range m.Content {
		if block.Type == ContentTypeText && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "")
}

// ToolCalls returns all tool-call blocks in the message.
func (m Message) ToolCalls() []tool.Call {
	var calls []tool.Call
	for _, block := range m.Content {
		if block.Type == ContentTypeToolCall && block.ToolCall != nil {
			calls = append(calls, *block.ToolCall)
		}
	}
	return calls
}

// ToolResults returns all tool-result blocks in the message.
func (m Message) ToolResults() []tool.Result {
	var results []tool.Result
	for _, block := range m.Content {
		if block.Type == ContentTypeToolResult && block.ToolResult != nil {
			results = append(results, *block.ToolResult)
		}
	}
	return results
}
