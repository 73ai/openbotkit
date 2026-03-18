package tools

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestParseStreamLine_Text(t *testing.T) {
	line := []byte(`{"type":"text","content":"hello"}`)
	evt := parseStreamLine(line)
	if evt.Type != "text" {
		t.Errorf("Type = %q", evt.Type)
	}
	if evt.Content != "hello" {
		t.Errorf("Content = %q", evt.Content)
	}
}

func TestParseStreamLine_Result(t *testing.T) {
	line := []byte(`{"type":"result","result":"final answer"}`)
	evt := parseStreamLine(line)
	if evt.Type != "result" {
		t.Errorf("Type = %q", evt.Type)
	}
	if evt.Content != "final answer" {
		t.Errorf("Content = %q", evt.Content)
	}
}

func TestParseStreamLine_ToolUse(t *testing.T) {
	line := []byte(`{"type":"tool_use","content":"bash"}`)
	evt := parseStreamLine(line)
	if evt.Type != "tool_use" {
		t.Errorf("Type = %q", evt.Type)
	}
}

func TestParseStreamLine_InvalidJSON(t *testing.T) {
	line := []byte(`not json at all`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("expected empty event for invalid JSON, got Type=%q", evt.Type)
	}
}

func TestParseStreamLine_GeminiMessage(t *testing.T) {
	line := []byte(`{"type":"message","role":"assistant","content":"Hello","delta":true}`)
	evt := parseStreamLine(line)
	if evt.Type != "text" {
		t.Errorf("Type = %q, want text", evt.Type)
	}
	if evt.Content != "Hello" {
		t.Errorf("Content = %q", evt.Content)
	}
}

func TestParseStreamLine_GeminiResult(t *testing.T) {
	line := []byte(`{"type":"result","status":"success","stats":{"total_tokens":100}}`)
	evt := parseStreamLine(line)
	if evt.Type != "result" {
		t.Errorf("Type = %q, want result", evt.Type)
	}
}

func TestParseStreamLine_GeminiUserMessage(t *testing.T) {
	line := []byte(`{"type":"message","role":"user","content":"prompt"}`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("user messages should be dropped, got Type=%q", evt.Type)
	}
}

func TestParseStreamLine_CodexItemCompleted(t *testing.T) {
	line := []byte(`{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"Hello"}}`)
	evt := parseStreamLine(line)
	if evt.Type != "text" {
		t.Errorf("Type = %q, want text", evt.Type)
	}
	if evt.Content != "Hello" {
		t.Errorf("Content = %q", evt.Content)
	}
}

func TestParseStreamLine_CodexReasoningIgnored(t *testing.T) {
	line := []byte(`{"type":"item.completed","item":{"type":"reasoning","text":"thinking..."}}`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("reasoning items should be dropped, got Type=%q", evt.Type)
	}
}

func TestParseStreamLine_CodexTurnCompleted(t *testing.T) {
	line := []byte(`{"type":"turn.completed","usage":{"input_tokens":100}}`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("turn.completed should be dropped, got Type=%q", evt.Type)
	}
}

func TestParseStreamLine_GeminiInit(t *testing.T) {
	line := []byte(`{"type":"init","session_id":"abc"}`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("init should be dropped, got Type=%q", evt.Type)
	}
}

func TestParseStreamLine_EmptyType(t *testing.T) {
	line := []byte(`{"content":"orphan"}`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("expected empty event for missing type, got Type=%q", evt.Type)
	}
}

func TestStreamRunner_BuildsClaudeStreamArgs(t *testing.T) {
	r := NewStreamRunner(AgentInfo{Kind: AgentClaude, Binary: "claude"})
	args := r.buildStreamArgs(runOptions{})
	want := []string{"--print", "--verbose", "--output-format", "stream-json", "--dangerously-skip-permissions"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, want[i])
		}
	}
}

func TestStreamRunner_BuildsGeminiStreamArgs(t *testing.T) {
	r := NewStreamRunner(AgentInfo{Kind: AgentGemini, Binary: "gemini"})
	args := r.buildStreamArgs(runOptions{})
	want := []string{"--approval-mode", "yolo", "-o", "stream-json"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, want[i])
		}
	}
}

func TestStreamRunner_BuildsCodexStreamArgs(t *testing.T) {
	r := NewStreamRunner(AgentInfo{Kind: AgentCodex, Binary: "codex"})
	args := r.buildStreamArgs(runOptions{})
	want := []string{"exec", "--json", "--full-auto"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, want[i])
		}
	}
}

func TestStreamRunner_BuildsClaudeStreamArgsWithBudget(t *testing.T) {
	r := NewStreamRunner(AgentInfo{Kind: AgentClaude, Binary: "claude"})
	args := r.buildStreamArgs(runOptions{maxBudgetUSD: 0.50})
	found := false
	for i, a := range args {
		if a == "--max-budget-usd" && i+1 < len(args) && args[i+1] == "0.50" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("args missing --max-budget-usd 0.50: %v", args)
	}
}

func TestStreamRunner_AccumulatesOutput(t *testing.T) {
	events := [][]byte{
		[]byte(`{"type":"text","content":"hello "}`),
		[]byte(`{"type":"text","content":"world"}`),
	}
	var accumulated string
	for _, line := range events {
		evt := parseStreamLine(line)
		if evt.Type == "text" || evt.Type == "result" {
			accumulated += evt.Content
		}
	}
	if accumulated != "hello world" {
		t.Errorf("accumulated = %q", accumulated)
	}
}

func TestStreamRunner_EmptyStream(t *testing.T) {
	events := [][]byte{}
	var accumulated string
	for _, line := range events {
		evt := parseStreamLine(line)
		if evt.Type == "text" || evt.Type == "result" {
			accumulated += evt.Content
		}
	}
	if accumulated != "" {
		t.Errorf("accumulated = %q, want empty", accumulated)
	}
}

func TestStreamRunner_RealClaude(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude not on PATH")
	}
	agents := DetectAgents()
	var info AgentInfo
	for _, a := range agents {
		if a.Kind == AgentClaude {
			info = a
			break
		}
	}
	r := NewStreamRunner(info)
	var eventCount int
	out, err := r.RunStream(context.Background(), "Say hello in exactly one word.", defaultDelegateTimeout, func(evt StreamEvent) {
		eventCount++
	})
	if err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
	if eventCount == 0 {
		t.Error("expected at least one event")
	}
}

func TestStreamRunner_RealGemini(t *testing.T) {
	if _, err := exec.LookPath("gemini"); err != nil {
		t.Skip("gemini not on PATH")
	}
	agents := DetectAgents()
	var info AgentInfo
	for _, a := range agents {
		if a.Kind == AgentGemini {
			info = a
			break
		}
	}
	r := NewStreamRunner(info)
	var eventCount int
	out, err := r.RunStream(context.Background(), "Say hello in exactly one word.", defaultDelegateTimeout, func(evt StreamEvent) {
		eventCount++
	})
	if err != nil {
		if strings.Contains(err.Error(), "Permission") || strings.Contains(err.Error(), "denied") || strings.Contains(err.Error(), "auth") {
			t.Skipf("gemini auth not configured: %v", err)
		}
		t.Fatalf("RunStream: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestStreamRunner_RealCodex(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex not on PATH")
	}
	agents := DetectAgents()
	var info AgentInfo
	for _, a := range agents {
		if a.Kind == AgentCodex {
			info = a
			break
		}
	}
	r := NewStreamRunner(info)
	var eventCount int
	out, err := r.RunStream(context.Background(), "Say hello in exactly one word.", defaultDelegateTimeout, func(evt StreamEvent) {
		eventCount++
	})
	if err != nil {
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "login") {
			t.Skipf("codex auth not configured: %v", err)
		}
		t.Fatalf("RunStream: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}
