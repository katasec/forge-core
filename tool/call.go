package tool

import "encoding/json"

// Call represents a request from the LLM to invoke a tool.
type Call struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Result represents the outcome of a tool invocation.
type Result struct {
	CallID  string `json:"call_id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// Error wraps a tool invocation failure.
type Error struct {
	CallID  string `json:"call_id"`
	Message string `json:"message"`
}
