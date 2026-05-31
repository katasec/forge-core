package forge

import (
	"context"
	"testing"
)

func TestSingleMiddleware(t *testing.T) {
	called := false
	mw := Middleware(func(next RunFunc) RunFunc {
		return func(ctx context.Context, req ProviderRequest) (*ProviderResponse, error) {
			called = true
			return next(ctx, req)
		}
	})

	inner := RunFunc(func(_ context.Context, _ ProviderRequest) (*ProviderResponse, error) {
		return &ProviderResponse{
			Messages:     []Message{AssistantText("ok")},
			FinishReason: FinishReasonStop,
		}, nil
	})

	run := mw(inner)
	resp, err := run(context.Background(), ProviderRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("middleware was not called")
	}
	if resp.Messages[0].Text() != "ok" {
		t.Errorf("Content = %q, want %q", resp.Messages[0].Text(), "ok")
	}
}

func TestMiddlewareCompositionOrder(t *testing.T) {
	var order []string

	makeMW := func(name string) Middleware {
		return func(next RunFunc) RunFunc {
			return func(ctx context.Context, req ProviderRequest) (*ProviderResponse, error) {
				order = append(order, name+"-before")
				resp, err := next(ctx, req)
				order = append(order, name+"-after")
				return resp, err
			}
		}
	}

	middlewares := []Middleware{makeMW("A"), makeMW("B"), makeMW("C")}

	inner := RunFunc(func(_ context.Context, _ ProviderRequest) (*ProviderResponse, error) {
		order = append(order, "provider")
		return &ProviderResponse{
			Messages:     []Message{AssistantText("ok")},
			FinishReason: FinishReasonStop,
		}, nil
	})

	// Apply middlewares per design spec: innermost-last.
	run := inner
	for i := len(middlewares) - 1; i >= 0; i-- {
		run = middlewares[i](run)
	}

	_, err := run(context.Background(), ProviderRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"A-before", "B-before", "C-before", "provider", "C-after", "B-after", "A-after"}
	if len(order) != len(expected) {
		t.Fatalf("got %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}
