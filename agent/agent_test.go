package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

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

// errorExecutor returns an error for specified tools.
type errorExecutor struct {
	successes map[string]string
	errors    map[string]string
	calls     []provider.ToolCall
}

func (m *errorExecutor) Execute(_ context.Context, call provider.ToolCall) (string, error) {
	m.calls = append(m.calls, call)
	if errMsg, ok := m.errors[call.Name]; ok {
		return "", fmt.Errorf("%s", errMsg)
	}
	if result, ok := m.successes[call.Name]; ok {
		return result, nil
	}
	return "", fmt.Errorf("unknown tool %q", call.Name)
}

func (m *errorExecutor) ToolSchemas() []provider.Tool {
	return []provider.Tool{
		{Name: "bash", Description: "Run a command", InputSchema: json.RawMessage(`{"type":"object"}`)},
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

func TestLoop_ScrubsToolOutput(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{
						ID: "c1", Name: "bash", Input: json.RawMessage(`{}`),
					}},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{
		"bash": "TOKEN=sk-secret-key-12345678",
	}}
	a := New(mp, "test-model", exec)

	_, err := a.Run(context.Background(), "show env")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The second request should contain the scrubbed tool result.
	if len(mp.requests) < 2 {
		t.Fatalf("expected 2 requests, got %d", len(mp.requests))
	}
	msgs := mp.requests[1].Messages
	// Last message should be the tool result.
	last := msgs[len(msgs)-1]
	content := last.Content[0].ToolResult.Content
	if strings.Contains(content, "sk-secret-key-12345678") {
		t.Errorf("tool output not scrubbed: %q", content)
	}
	if !strings.Contains(content, "****") {
		t.Errorf("expected redacted content, got: %q", content)
	}
}

func TestLoop_ScrubsToolError(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{
						ID: "c1", Name: "bash", Input: json.RawMessage(`{}`),
					}},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &errorExecutor{
		errors: map[string]string{"bash": "failed: password=supersecret123"},
	}
	a := New(mp, "test-model", exec)

	_, err := a.Run(context.Background(), "try")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	msgs := mp.requests[1].Messages
	last := msgs[len(msgs)-1]
	content := last.Content[0].ToolResult.Content
	if strings.Contains(content, "supersecret123") {
		t.Errorf("tool error not scrubbed: %q", content)
	}
	if !last.Content[0].ToolResult.IsError {
		t.Error("expected IsError=true")
	}
}

func TestLoop_ToolErrorIncludesOutput(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{
						ID: "c1", Name: "bash", Input: json.RawMessage(`{}`),
					}},
				},
				StopReason: provider.StopToolUse,
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &outputAndErrorExecutor{
		output: "Usage: gws calendar events list [flags]",
		err:    "gws: exit status 1",
	}
	a := New(mp, "test-model", exec)

	_, err := a.Run(context.Background(), "try")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	msgs := mp.requests[1].Messages
	last := msgs[len(msgs)-1]
	content := last.Content[0].ToolResult.Content
	if !strings.Contains(content, "Usage: gws calendar events list") {
		t.Errorf("tool result should include command output, got: %q", content)
	}
	if !strings.Contains(content, "exit status 1") {
		t.Errorf("tool result should include error, got: %q", content)
	}
	if !last.Content[0].ToolResult.IsError {
		t.Error("expected IsError=true")
	}
}

// outputAndErrorExecutor returns both output and an error (like gws runner does).
type outputAndErrorExecutor struct {
	output string
	err    string
}

func (m *outputAndErrorExecutor) Execute(_ context.Context, _ provider.ToolCall) (string, error) {
	return m.output, fmt.Errorf("%s", m.err)
}

func (m *outputAndErrorExecutor) ToolSchemas() []provider.Tool {
	return []provider.Tool{
		{Name: "bash", Description: "Run a command", InputSchema: json.RawMessage(`{"type":"object"}`)},
	}
}

func TestLoop_ProviderChatError(t *testing.T) {
	mp := &mockProvider{responses: nil} // no responses = error on first call
	exec := &mockExecutor{results: map[string]string{}}
	a := New(mp, "test-model", exec)

	_, err := a.Run(context.Background(), "hi")
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if !strings.Contains(err.Error(), "chat (iteration 0)") {
		t.Errorf("error = %q, expected chat iteration error", err.Error())
	}
}

func TestLoop_CompactsHistory(t *testing.T) {
	// Build a provider that does one tool call per iteration for 15 rounds, then ends.
	var responses []*provider.ChatResponse
	for i := range 15 {
		responses = append(responses, &provider.ChatResponse{
			Content: []provider.ContentBlock{
				{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{
					ID: fmt.Sprintf("c%d", i), Name: "bash", Input: json.RawMessage(`{}`),
				}},
			},
			StopReason: provider.StopToolUse,
		})
	}
	responses = append(responses, &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
		StopReason: provider.StopEndTurn,
	})

	mp := &mockProvider{responses: responses}
	exec := &mockExecutor{results: map[string]string{"bash": "ok"}}
	// Without compaction, history would be 1 user + 15*(assistant+result) + final assistant = 32 messages.
	// With maxHistory=10, compaction fires repeatedly, keeping history bounded.
	a := New(mp, "test-model", exec, WithMaxHistory(10), WithMaxIterations(20))

	_, err := a.Run(context.Background(), "go")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// History should have been compacted (not 32 messages).
	if len(a.history) > 22 {
		t.Errorf("history not compacted: len=%d, want <=22", len(a.history))
	}
}

type mockUsageRecorder struct {
	records []provider.Usage
}

func (m *mockUsageRecorder) RecordUsage(model string, usage provider.Usage) {
	m.records = append(m.records, usage)
}

func TestLoop_UsageRecorder(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{ID: "c1", Name: "bash", Input: json.RawMessage(`{}`)}}},
				StopReason: provider.StopToolUse,
				Usage:      provider.Usage{InputTokens: 100, OutputTokens: 50, CacheReadTokens: 80},
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
				StopReason: provider.StopEndTurn,
				Usage:      provider.Usage{InputTokens: 200, OutputTokens: 30},
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{"bash": "ok"}}
	recorder := &mockUsageRecorder{}
	a := New(mp, "test-model", exec, WithUsageRecorder(recorder))

	_, err := a.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(recorder.records) != 2 {
		t.Fatalf("recorded %d usages, want 2", len(recorder.records))
	}
	if recorder.records[0].CacheReadTokens != 80 {
		t.Errorf("first call CacheReadTokens = %d, want 80", recorder.records[0].CacheReadTokens)
	}
	if recorder.records[1].InputTokens != 200 {
		t.Errorf("second call InputTokens = %d, want 200", recorder.records[1].InputTokens)
	}
}

func TestLoop_SystemBlocks(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "ok"}},
				StopReason: provider.StopEndTurn,
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{}}
	blocks := []provider.SystemBlock{
		{Text: "block1", CacheControl: &provider.CacheControl{Type: "ephemeral"}},
		{Text: "block2"},
	}
	a := New(mp, "test-model", exec, WithSystemBlocks(blocks))

	_, err := a.Run(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(mp.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mp.requests))
	}
	req := mp.requests[0]
	if len(req.SystemBlocks) != 2 {
		t.Fatalf("SystemBlocks length = %d, want 2", len(req.SystemBlocks))
	}
	if req.SystemBlocks[0].Text != "block1" {
		t.Errorf("block 0 text = %q", req.SystemBlocks[0].Text)
	}
	if req.SystemBlocks[0].CacheControl == nil {
		t.Error("block 0 should have cache_control")
	}
	if req.SystemBlocks[1].CacheControl != nil {
		t.Error("block 1 should not have cache_control")
	}
}

func TestLoop_RateLimiterContextCancel(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{Content: []provider.ContentBlock{{Type: provider.ContentText, Text: "ok"}}, StopReason: provider.StopEndTurn},
		},
	}
	exec := &mockExecutor{results: map[string]string{}}

	// Create agent with extremely low rate limit (1/hour).
	a := New(mp, "test-model", exec, WithRateLimit(1))

	// First call uses burst, should succeed.
	_, err := a.Run(context.Background(), "first")
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	// Exhaust remaining burst by calling multiple times.
	for range 9 {
		mp.idx = 0
		mp.requests = nil
		a.history = nil
		_, _ = a.Run(context.Background(), "burst")
	}

	// Now cancel context; should fail on rate limiter.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mp.idx = 0
	mp.requests = nil
	a.history = nil
	_, err = a.Run(ctx, "should fail")
	if err == nil {
		t.Fatal("expected rate limiter error")
	}
	if !strings.Contains(err.Error(), "rate limiter") {
		t.Errorf("error = %q, expected rate limiter error", err.Error())
	}
}

func TestLoop_TokenBasedCompaction(t *testing.T) {
	ms := &mockSummarizer{result: "conversation summary"}
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content: []provider.ContentBlock{
					{Type: provider.ContentToolUse, ToolCall: &provider.ToolCall{
						ID: "c1", Name: "bash", Input: json.RawMessage(`{}`),
					}},
				},
				StopReason: provider.StopToolUse,
				Usage:      provider.Usage{InputTokens: 70000},
			},
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "Done"}},
				StopReason: provider.StopEndTurn,
				Usage:      provider.Usage{InputTokens: 70000},
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{"bash": "ok"}}
	a := New(mp, "test-model", exec,
		WithContextWindow(200000),
		WithCompactionThreshold(0.30),
		WithSummarizer(ms),
		WithMaxIterations(10),
	)

	result, err := a.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result != "Done" {
		t.Errorf("result = %q, want 'Done'", result)
	}

	// After the first LLM call returns 70000 tokens (> 200000 * 0.30 = 60000),
	// the second iteration should trigger compactHistory which calls the summarizer.
	if !ms.called {
		t.Error("summarizer should have been called during compaction")
	}

	// Verify the summary was injected into the messages sent to the provider.
	if len(mp.requests) < 2 {
		t.Fatalf("expected at least 2 requests, got %d", len(mp.requests))
	}
	secondReqMsgs := mp.requests[1].Messages
	foundSummary := false
	for _, m := range secondReqMsgs {
		for _, c := range m.Content {
			if strings.Contains(c.Text, "conversation summary") {
				foundSummary = true
			}
		}
	}
	if !foundSummary {
		t.Error("second request should contain the compaction summary")
	}
}

func TestLoop_TracksLastInputTokens(t *testing.T) {
	mp := &mockProvider{
		responses: []*provider.ChatResponse{
			{
				Content:    []provider.ContentBlock{{Type: provider.ContentText, Text: "ok"}},
				StopReason: provider.StopEndTurn,
				Usage:      provider.Usage{InputTokens: 42000},
			},
		},
	}
	exec := &mockExecutor{results: map[string]string{}}
	a := New(mp, "test-model", exec)

	_, err := a.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if a.lastInputTokens != 42000 {
		t.Errorf("lastInputTokens = %d, want 42000", a.lastInputTokens)
	}
}
