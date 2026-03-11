package tools

import (
	"context"
	"os/exec"
	"testing"
	"time"
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

func TestParseStreamLine_EmptyType(t *testing.T) {
	line := []byte(`{"content":"orphan"}`)
	evt := parseStreamLine(line)
	if evt.Type != "" {
		t.Errorf("expected empty event for missing type, got Type=%q", evt.Type)
	}
}

func TestStreamRunner_BuildsClaudeStreamArgs(t *testing.T) {
	r := NewStreamRunner(AgentInfo{Kind: AgentClaude, Binary: "/usr/local/bin/claude"})
	args := r.buildStreamArgs(runOptions{})
	want := []string{"--print", "--verbose", "--output-format", "stream-json"}
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
	r := NewStreamRunner(AgentInfo{Kind: AgentGemini, Binary: "/usr/local/bin/gemini"})
	args := r.buildStreamArgs(runOptions{})
	want := []string{"-o", "stream-json"}
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
	r := NewStreamRunner(AgentInfo{Kind: AgentClaude, Binary: "/usr/local/bin/claude"})
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
	out, err := r.RunStream(context.Background(), "Say hello in exactly one word.", 30*time.Second, func(evt StreamEvent) {
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
