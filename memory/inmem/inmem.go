package inmem

import (
	"context"
	"sync"

	"github.com/katasec/forge-core/message"
)

// Store is a thread-safe in-memory memory store.
type Store struct {
	mu   sync.RWMutex
	data map[string][]message.Message
}

// New creates an empty in-memory store.
func New() *Store {
	return &Store{
		data: make(map[string][]message.Message),
	}
}

// Load returns a copy of the stored messages for the given conversation.
func (s *Store) Load(_ context.Context, conversationID string) ([]message.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs, ok := s.data[conversationID]
	if !ok {
		return nil, nil
	}

	return cloneMessages(msgs), nil
}

// Save replaces the entire message history for the given conversation.
func (s *Store) Save(_ context.Context, conversationID string, messages []message.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[conversationID] = cloneMessages(messages)
	return nil
}

// Clear deletes the conversation history.
func (s *Store) Clear(_ context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, conversationID)
	return nil
}

func cloneMessages(messages []message.Message) []message.Message {
	cp := make([]message.Message, len(messages))
	for i, msg := range messages {
		cp[i] = msg
		cp[i].Content = cloneContentBlocks(msg.Content)
	}
	return cp
}

func cloneContentBlocks(blocks []message.ContentBlock) []message.ContentBlock {
	cp := make([]message.ContentBlock, len(blocks))
	for i, block := range blocks {
		cp[i] = block
		if block.Image != nil {
			image := *block.Image
			if image.Data != nil {
				image.Data = append([]byte(nil), image.Data...)
			}
			cp[i].Image = &image
		}
		if block.ToolCall != nil {
			call := *block.ToolCall
			call.Arguments = append([]byte(nil), call.Arguments...)
			cp[i].ToolCall = &call
		}
		if block.ToolResult != nil {
			result := *block.ToolResult
			cp[i].ToolResult = &result
		}
		if block.Metadata != nil {
			metadata := make(map[string]any, len(block.Metadata))
			for k, v := range block.Metadata {
				metadata[k] = v
			}
			cp[i].Metadata = metadata
		}
	}
	return cp
}
