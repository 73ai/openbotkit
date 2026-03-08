package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
)

// mockProvider returns scripted responses in sequence.
type mockProvider struct {
	responses []*provider.ChatResponse
	requests  []provider.ChatRequest
	idx       int
}

func (m *mockProvider) Chat(_ context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.requests = append(m.requests, req)
	if m.idx >= len(m.responses) {
		return nil, fmt.Errorf("no more responses (called %d times)", m.idx+1)
	}
	resp := m.responses[m.idx]
	m.idx++
	return resp, nil
}

func (m *mockProvider) StreamChat(_ context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

// mockExecutor records tool calls and returns canned results.
type mockExecutor struct {
	results map[string]string
	calls   []provider.ToolCall
}

func (m *mockExecutor) Execute(_ context.Context, call provider.ToolCall) (string, error) {
	m.calls = append(m.calls, call)
	if result, ok := m.results[call.Name]; ok {
		return result, nil
	}
	return "", fmt.Errorf("unknown tool %q", call.Name)
}

func (m *mockExecutor) ToolSchemas() []provider.Tool {
	return []provider.Tool{
		{
			Name:        "bash",
			Description: "Run a command",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"}}}`),
		},
	}
}

func TestLoop_SimpleText(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Hello!"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{}}
	agent := New(mp, "test-model", exec)

	result, err := agent.Run(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "Hello!" {
		t.Errorf("result = %q, want %q", result, "Hello!")
	}
	if len(mp.requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(mp.requests))
	}
}

func TestLoop_SingleToolCall(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentText, Text: "Let me run that."},
					{
						Type: provider.ContentToolUse,
						ToolCall: &provider.ToolCall{
							ID:    "call_1",
							Name:  "bash",
							Input: json.RawMessage(`{"command":"echo hello"}`),
						},
					},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "The output is: hello"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{"bash": "hello\n"}}
	agent := New(mp, "test-model", exec)

	result, err := agent.Run(context.Background(), "run echo hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "The output is: hello" {
		t.Errorf("result = %q", result)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(exec.calls))
	}
	if exec.calls[0].Name != "bash" {
		t.Errorf("tool name = %q", exec.calls[0].Name)
	}
}

func TestLoop_MultiToolSequence(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{ID: "c1", Name: "bash", Input: json.RawMessage(`{}`)}},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{ID: "c2", Name: "bash", Input: json.RawMessage(`{}`)}},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{ID: "c3", Name: "bash", Input: json.RawMessage(`{}`)}},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{"bash": "ok"}}
	agent := New(mp, "test-model", exec)

	result, err := agent.Run(context.Background(), "do stuff")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "Done" {
		t.Errorf("result = %q", result)
	}
	if len(exec.calls) != 3 {
		t.Errorf("expected 3 tool calls, got %d", len(exec.calls))
	}
}

func TestLoop_MaxIterations(t *testing.T) {
	// Provider always returns tool_use — should stop at max iterations.
	alwaysToolUse := &provider.ChatResponse{
		Content: []provider.ContentBlock{
			{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{ID: "c1", Name: "bash", Input: json.RawMessage(`{}`)}},
		},
		StopReason: provider.StopToolUse,
	}

	// Create enough responses for max iterations.
	responses := make([]*provider.ChatResponse, 5)
	for i := range responses {
		responses[i] = alwaysToolUse
	}

	mp := &mockProvider{responses: responses}
	exec := &mockExecutor{results: map[string]string{"bash": "ok"}}
	agent := New(mp, "test-model", exec, WithMaxIterations(5))

	_, err := agent.Run(context.Background(), "infinite loop")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "max iterations (5) reached" {
		t.Errorf("error = %q", got)
	}
}
