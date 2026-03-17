package telegram

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/73ai/openbotkit/provider"
)

type stubExecutor struct {
	result  string
	schemas []provider.Tool
	calls   []provider.ToolCall
}

func (s *stubExecutor) Execute(_ context.Context, call provider.ToolCall) (string, error) {
	s.calls = append(s.calls, call)
	return s.result, nil
}

func (s *stubExecutor) ToolSchemas() []provider.Tool { return s.schemas }

func TestNotifyingExecutor_CallsOnToolStart(t *testing.T) {
	var called string
	delegate := &stubExecutor{result: "ok"}
	ne := &notifyingExecutor{
		delegate:    delegate,
		onToolStart: func(name string) { called = name },
	}

	_, err := ne.Execute(context.Background(), provider.ToolCall{
		ID: "c1", Name: "web_search", Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if called != "web_search" {
		t.Errorf("onToolStart called with %q, want %q", called, "web_search")
	}
}

func TestNotifyingExecutor_DelegatesToOriginal(t *testing.T) {
	delegate := &stubExecutor{result: "the result"}
	ne := &notifyingExecutor{
		delegate:    delegate,
		onToolStart: func(string) {},
	}

	result, err := ne.Execute(context.Background(), provider.ToolCall{
		ID: "c1", Name: "bash", Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "the result" {
		t.Errorf("result = %q, want %q", result, "the result")
	}
	if len(delegate.calls) != 1 {
		t.Fatalf("expected 1 delegate call, got %d", len(delegate.calls))
	}
}

func TestNotifyingExecutor_ToolSchemas_PassThrough(t *testing.T) {
	schemas := []provider.Tool{
		{Name: "bash", Description: "Run a command", InputSchema: json.RawMessage(`{}`)},
		{Name: "search", Description: "Search", InputSchema: json.RawMessage(`{}`)},
	}
	delegate := &stubExecutor{schemas: schemas}
	ne := &notifyingExecutor{delegate: delegate, onToolStart: func(string) {}}

	got := ne.ToolSchemas()
	if len(got) != 2 {
		t.Fatalf("ToolSchemas returned %d, want 2", len(got))
	}
	if got[0].Name != "bash" || got[1].Name != "search" {
		t.Errorf("schemas = %v", got)
	}
}
