package runtime

import (
	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/provider"
	"github.com/katasec/forge-core/tool"
)

// AgentRequest is the input to Agent.Run.
type AgentRequest struct {
	ConversationID string            `json:"conversation_id"`
	Messages       []message.Message `json:"messages"`
}

// AgentResponse is the output of Agent.Run.
type AgentResponse struct {
	ConversationID string                `json:"conversation_id"`
	Messages       []message.Message     `json:"messages"`
	FinishReason   provider.FinishReason `json:"finish_reason"`
	Usage          provider.TokenUsage   `json:"usage"`
	Errors         []tool.Error          `json:"errors,omitempty"`
}

// LastText returns the latest assistant text in the response.
func (r *AgentResponse) LastText() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		msg := r.Messages[i]
		if msg.Role == message.RoleAssistant && msg.Text() != "" {
			return msg.Text()
		}
	}
	return ""
}

type ErrorPolicy string

const (
	ErrorPolicyStop     ErrorPolicy = "stop"
	ErrorPolicyContinue ErrorPolicy = "continue"
)
