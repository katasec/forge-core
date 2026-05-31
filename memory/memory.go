package memory

import (
	"context"

	"github.com/katasec/forge-core/message"
)

// Store persists conversation message history.
type Store interface {
	Load(ctx context.Context, conversationID string) ([]message.Message, error)
	Save(ctx context.Context, conversationID string, messages []message.Message) error
	Clear(ctx context.Context, conversationID string) error
}
