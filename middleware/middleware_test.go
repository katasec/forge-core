package middleware

import (
	"context"
	"testing"

	"github.com/katasec/forge-core/message"
	"github.com/katasec/forge-core/provider"
)

func TestSingleMiddleware(t *testing.T) {
	called := false
	mw := Middleware(func(next RunFunc) RunFunc {
		return func(ctx context.Context, req provider.Request) (*provider.Response, error) {
			called = true
			return next(ctx, req)
		}
	})

	inner := RunFunc(func(_ context.Context, _ provider.Request) (*provider.Response, error) {
		return &provider.Response{
			Messages:     []message.Message{message.AssistantText("ok")},
			FinishReason: provider.FinishReasonStop,
		}, nil
	})

	run := mw(inner)
	resp, err := run(context.Background(), provider.Request{})
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

	middlewares := []Middleware{
		recordingMiddleware("A", &order),
		recordingMiddleware("B", &order),
		recordingMiddleware("C", &order),
	}

	run := composeForTest(providerRunFunc(&order), middlewares)

	_, err := run(context.Background(), provider.Request{})
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

func recordingMiddleware(name string, order *[]string) Middleware {
	return func(next RunFunc) RunFunc {
		return func(ctx context.Context, req provider.Request) (*provider.Response, error) {
			*order = append(*order, name+"-before")
			resp, err := next(ctx, req)
			*order = append(*order, name+"-after")
			return resp, err
		}
	}
}

func providerRunFunc(order *[]string) RunFunc {
	return func(_ context.Context, _ provider.Request) (*provider.Response, error) {
		*order = append(*order, "provider")
		return &provider.Response{
			Messages:     []message.Message{message.AssistantText("ok")},
			FinishReason: provider.FinishReasonStop,
		}, nil
	}
}

func composeForTest(run RunFunc, middlewares []Middleware) RunFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		run = middlewares[i](run)
	}
	return run
}
