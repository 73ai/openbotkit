package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/provider"
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

func (m *mockProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

// stubTool is a minimal tool that always returns a fixed output.
type stubTool struct {
	name   string
	output string
}

func (s *stubTool) Name() string                                                          { return s.name }
func (s *stubTool) Description() string                                                   { return "stub" }
func (s *stubTool) InputSchema() json.RawMessage                                          { return json.RawMessage(`{"type":"object"}`) }
func (s *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return s.output, nil }

func newSubagentTool(mp *mockProvider) *tools.SubagentTool {
	return tools.NewSubagentTool(tools.SubagentConfig{
		Provider: mp,
		Model:    "test-model",
		ToolFactory: func() *tools.Registry {
			r := tools.NewRegistry()
			r.Register(&stubTool{name: "bash", output: "ok"})
			return r
		},
		System: "You are a test sub-agent.",
	})
}

func TestSubagentTool_Name(t *testing.T) {
	mp := &mockProvider{}
	tool := newSubagentTool(mp)
	if got := tool.Name(); got != "subagent" {
		t.Errorf("Name() = %q, want %q", got, "subagent")
	}
}

func TestSubagentTool_SimpleTask(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "The answer is 42."}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	tool := newSubagentTool(mp)

	input := json.RawMessage(`{"task": "What is the answer?"}`)
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "The answer is 42." {
		t.Errorf("result = %q, want %q", result, "The answer is 42.")
	}
	if len(mp.requests) != 1 {
		t.Errorf("expected 1 provider request, got %d", len(mp.requests))
	}
	if mp.requests[0].System != "You are a test sub-agent." {
		t.Errorf("system = %q", mp.requests[0].System)
	}
}

func TestSubagentTool_ChildUsesTools(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
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
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done: hello"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	tool := newSubagentTool(mp)

	input := json.RawMessage(`{"task": "Run echo hello"}`)
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "Done: hello" {
		t.Errorf("result = %q", result)
	}
	if len(mp.requests) != 2 {
		t.Errorf("expected 2 provider requests (tool call + final), got %d", len(mp.requests))
	}
}

func TestSubagentTool_ChildMaxIterations(t *testing.T) {
	alwaysToolUse := &provider.ChatResponse{
		Content: []provider.ContentBlock{
			{
				Type: provider.ContentToolUse,
				ToolCall: &provider.ToolCall{
					ID:    "c1",
					Name:  "bash",
					Input: json.RawMessage(`{"command":"true"}`),
				},
			},
		},
		StopReason: provider.StopToolUse,
	}

	responses := make([]*provider.ChatResponse, defaultChildMaxIter())
	for i := range responses {
		responses[i] = alwaysToolUse
	}

	mp := &mockProvider{responses: responses}
	tool := newSubagentTool(mp)

	input := json.RawMessage(`{"task": "infinite loop"}`)
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected max iterations error, got nil")
	}
	if !strings.Contains(err.Error(), "max iterations") {
		t.Errorf("error = %q, expected max iterations error", err.Error())
	}
}

func TestSubagentTool_NoRecursion(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "ok"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	tool := newSubagentTool(mp)

	input := json.RawMessage(`{"task": "test"}`)
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Check that the child's tool schemas do not include "subagent".
	childTools := mp.requests[0].Tools
	for _, ct := range childTools {
		if ct.Name == "subagent" {
			t.Error("child registry should not contain 'subagent' tool")
		}
	}
}

// ctxAwareMockProvider checks context before returning.
type ctxAwareMockProvider struct{ mockProvider }

func (m *ctxAwareMockProvider) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.mockProvider.Chat(ctx, req)
}

func TestSubagentTool_ContextCancellation(t *testing.T) {
	mp := &ctxAwareMockProvider{mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "ok"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}}
	tool := tools.NewSubagentTool(tools.SubagentConfig{
		Provider: mp,
		Model:    "test-model",
		ToolFactory: func() *tools.Registry {
			return tools.NewRegistry()
		},
		System: "test",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := json.RawMessage(`{"task": "test"}`)
	_, err := tool.Execute(ctx, input)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestSubagentTool_EmptyTask(t *testing.T) {
	mp := &mockProvider{}
	tool := newSubagentTool(mp)

	input := json.RawMessage(`{"task": ""}`)
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty task")
	}
	if !strings.Contains(err.Error(), "task is required") {
		t.Errorf("error = %q, expected 'task is required'", err.Error())
	}
}

// defaultChildMaxIter returns the default max iterations for child agents.
func defaultChildMaxIter() int { return 10 }
