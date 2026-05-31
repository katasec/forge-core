package message

import "testing"

func TestUserMessage(t *testing.T) {
	msg := UserText("hello")
	if msg.Role != RoleUser {
		t.Fatalf("Role = %q, want %q", msg.Role, RoleUser)
	}
	if msg.Text() != "hello" {
		t.Fatalf("Content = %q, want hello", msg.Text())
	}
}
