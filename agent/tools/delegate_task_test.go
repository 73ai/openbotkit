package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func setupDelegateTest(t *testing.T, approveAll bool) (*DelegateTaskTool, *mockInteractor, *mockAgentRunner) {
	t.Helper()
	inter := &mockInteractor{approveAll: approveAll}
	runner := &mockAgentRunner{output: "research result"}
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "/usr/local/bin/claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
	})
	tool.runners[AgentClaude] = runner
	return tool, inter, runner
}

func TestDelegateTask_Metadata(t *testing.T) {
	tool, _, _ := setupDelegateTest(t, true)
	if tool.Name() != "delegate_task" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
	if !json.Valid(tool.InputSchema()) {
		t.Error("InputSchema() is not valid JSON")
	}
	// Verify agent enum is present in schema.
	schema := string(tool.InputSchema())
	if !strings.Contains(schema, `"claude"`) {
		t.Errorf("schema missing claude enum: %s", schema)
	}
}

func TestDelegateTask_Approved(t *testing.T) {
	tool, inter, runner := setupDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: "research Go 1.23"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(inter.approvals) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(inter.approvals))
	}
	if !strings.Contains(inter.approvals[0], "claude") {
		t.Errorf("approval missing agent name: %q", inter.approvals[0])
	}
	if !runner.called {
		t.Error("runner was not called")
	}
	if runner.prompt != "research Go 1.23" {
		t.Errorf("prompt = %q", runner.prompt)
	}
	if result != "research result" {
		t.Errorf("result = %q", result)
	}
	if len(inter.notified) != 1 || inter.notified[0] != "Done." {
		t.Errorf("notified = %v", inter.notified)
	}
}

func TestDelegateTask_Denied(t *testing.T) {
	tool, inter, runner := setupDelegateTest(t, false)
	input, _ := json.Marshal(delegateTaskInput{Task: "research Go 1.23"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q, want %q", result, "denied_by_user")
	}
	if runner.called {
		t.Error("runner should not have been called")
	}
	if len(inter.notified) != 1 || inter.notified[0] != "Action not performed." {
		t.Errorf("notified = %v", inter.notified)
	}
}

func TestDelegateTask_ApprovalError(t *testing.T) {
	tool, inter, _ := setupDelegateTest(t, false)
	inter.approveErr = errors.New("connection lost")
	input, _ := json.Marshal(delegateTaskInput{Task: "research"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, inter.approveErr) {
		t.Errorf("error = %v, want %v", err, inter.approveErr)
	}
}

func TestDelegateTask_AutoSelectsFirst(t *testing.T) {
	tool, _, runner := setupDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: "analyze data"})
	_, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !runner.called {
		t.Error("first agent (claude) should have been used")
	}
}

func TestDelegateTask_ExplicitAgent(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	claudeRunner := &mockAgentRunner{output: "claude output"}
	geminiRunner := &mockAgentRunner{output: "gemini output"}
	agents := []AgentInfo{
		{Kind: AgentClaude, Binary: "/usr/local/bin/claude"},
		{Kind: AgentGemini, Binary: "/usr/local/bin/gemini"},
	}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
	})
	tool.runners[AgentClaude] = claudeRunner
	tool.runners[AgentGemini] = geminiRunner

	input, _ := json.Marshal(delegateTaskInput{Task: "research", Agent: "gemini"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if claudeRunner.called {
		t.Error("claude runner should not have been called")
	}
	if !geminiRunner.called {
		t.Error("gemini runner should have been called")
	}
	if result != "gemini output" {
		t.Errorf("result = %q", result)
	}
}

func TestDelegateTask_UnknownAgent(t *testing.T) {
	tool, _, _ := setupDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: "research", Agent: "codex"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(err.Error(), `"codex" not available`) {
		t.Errorf("error = %q", err.Error())
	}
}

func TestDelegateTask_EmptyTask(t *testing.T) {
	tool, inter, runner := setupDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: ""})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty task")
	}
	if !strings.Contains(err.Error(), "task is required") {
		t.Errorf("error = %q", err.Error())
	}
	if len(inter.approvals) != 0 {
		t.Error("no approval should be requested for empty task")
	}
	if runner.called {
		t.Error("runner should not be called for empty task")
	}
}

func TestDelegateTask_RunnerError(t *testing.T) {
	tool, _, runner := setupDelegateTest(t, true)
	runner.output = ""
	runner.err = errors.New("CLI crashed")
	input, _ := json.Marshal(delegateTaskInput{Task: "analyze"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error from runner")
	}
	if !strings.Contains(err.Error(), "CLI crashed") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestDelegateTask_Timeout(t *testing.T) {
	tool, _, runner := setupDelegateTest(t, true)
	runner.output = ""
	runner.err = context.DeadlineExceeded
	input, _ := json.Marshal(delegateTaskInput{Task: "slow task"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDelegateTask_ApprovalDescription(t *testing.T) {
	tool, inter, _ := setupDelegateTest(t, true)
	longTask := strings.Repeat("a", 200)
	input, _ := json.Marshal(delegateTaskInput{Task: longTask})
	tool.Execute(context.Background(), input)
	if len(inter.approvals) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(inter.approvals))
	}
	desc := inter.approvals[0]
	if !strings.HasPrefix(desc, "Delegate to claude: ") {
		t.Errorf("approval desc = %q", desc)
	}
	// truncateUTF8 truncates at 80 runes + "..."
	if len(desc) > len("Delegate to claude: ")+80+3+5 {
		t.Errorf("approval desc too long: %d chars", len(desc))
	}
}
