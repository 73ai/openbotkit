package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncMockInteractor is a thread-safe mock for async tests.
type syncMockInteractor struct {
	mu         sync.Mutex
	notified   []string
	approvals  []string
	approveAll bool
	approveErr error
	notifyCh   chan string // signals each Notify call
}

func newSyncMockInteractor(approveAll bool) *syncMockInteractor {
	return &syncMockInteractor{
		approveAll: approveAll,
		notifyCh:   make(chan string, 10),
	}
}

func (m *syncMockInteractor) Notify(msg string) error {
	m.mu.Lock()
	m.notified = append(m.notified, msg)
	m.mu.Unlock()
	m.notifyCh <- msg
	return nil
}

func (m *syncMockInteractor) NotifyLink(text, url string) error { return nil }

func (m *syncMockInteractor) RequestApproval(desc string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.approvals = append(m.approvals, desc)
	if m.approveErr != nil {
		return false, m.approveErr
	}
	return m.approveAll, nil
}

func (m *syncMockInteractor) getNotified() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.notified))
	copy(cp, m.notified)
	return cp
}

func setupDelegateTest(t *testing.T, approveAll bool) (*DelegateTaskTool, *mockInteractor, *mockAgentRunner) {
	t.Helper()
	inter := &mockInteractor{approveAll: approveAll}
	runner := &mockAgentRunner{output: "research result"}
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
	})
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil // disable streaming in tests
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
		{Kind: AgentClaude, Binary: "claude"},
		{Kind: AgentGemini, Binary: "gemini"},
	}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
	})
	tool.runners[AgentClaude] = claudeRunner
	tool.runners[AgentGemini] = geminiRunner
	tool.streamRunners[AgentClaude] = nil
	tool.streamRunners[AgentGemini] = nil

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

func setupAsyncDelegateTest(t *testing.T, approveAll bool) (*DelegateTaskTool, *syncMockInteractor, *blockingAgentRunner) {
	t.Helper()
	inter := newSyncMockInteractor(approveAll)
	runner := newBlockingRunner("async result", nil)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil // disable streaming, use mock runner
	return tool, inter, runner
}

func TestDelegateTask_AsyncReturnsImmediately(t *testing.T) {
	tool, _, runner := setupAsyncDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: "research async", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Should return immediately with task_id and status.
	var resp struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.TaskID == "" {
		t.Error("task_id is empty")
	}
	if resp.Status != "running" {
		t.Errorf("status = %q, want running", resp.Status)
	}
	// Release the runner so goroutine can finish.
	close(runner.release)
	// Wait for goroutine.
	<-time.After(100 * time.Millisecond)
}

func TestDelegateTask_AsyncCompleted(t *testing.T) {
	tool, inter, runner := setupAsyncDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: "research", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var resp struct{ TaskID string `json:"task_id"` }
	json.Unmarshal([]byte(result), &resp)

	// Wait for goroutine to start, then release.
	<-runner.called
	close(runner.release)
	// Wait for completion notification via channel.
	select {
	case <-inter.notifyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}

	rec, ok := tool.tracker.Get(resp.TaskID)
	if !ok {
		t.Fatal("task not found in tracker")
	}
	if rec.Status != TaskCompleted {
		t.Errorf("status = %q, want completed", rec.Status)
	}
	if rec.Output != "async result" {
		t.Errorf("output = %q", rec.Output)
	}
	// Check completion notification.
	found := false
	for _, n := range inter.getNotified() {
		if strings.Contains(n, "Task completed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing completion notification, got: %v", inter.getNotified())
	}
}

func TestDelegateTask_AsyncFailed(t *testing.T) {
	inter := newSyncMockInteractor(true)
	runner := newBlockingRunner("", errors.New("agent crashed"))
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "failing task", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var resp struct{ TaskID string `json:"task_id"` }
	json.Unmarshal([]byte(result), &resp)

	<-runner.called
	close(runner.release)
	// Wait for failure notification via channel.
	select {
	case <-inter.notifyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}

	rec, ok := tracker.Get(resp.TaskID)
	if !ok {
		t.Fatal("task not found")
	}
	if rec.Status != TaskFailed {
		t.Errorf("status = %q, want failed", rec.Status)
	}
	found := false
	for _, n := range inter.getNotified() {
		if strings.Contains(n, "Task failed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing failure notification, got: %v", inter.getNotified())
	}
}

func TestDelegateTask_AsyncMaxConcurrent(t *testing.T) {
	inter := newSyncMockInteractor(true)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}

	// Pre-fill tracker to max.
	tracker.Start("t1", "task1", AgentClaude)
	tracker.Start("t2", "task2", AgentClaude)
	tracker.Start("t3", "task3", AgentClaude)

	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	runner := &mockAgentRunner{output: "result"}
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "4th task", Async: true})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for max concurrent")
	}
	if !strings.Contains(err.Error(), "concurrent") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestDelegateTask_AsyncWithoutTracker(t *testing.T) {
	// Tracker is nil, async=true should fall back to sync.
	tool, _, runner := setupDelegateTest(t, true)
	input, _ := json.Marshal(delegateTaskInput{Task: "sync fallback", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !runner.called {
		t.Error("runner should have been called synchronously")
	}
	if result != "research result" {
		t.Errorf("result = %q", result)
	}
}

func TestDelegateTask_SyncIgnoresTracker(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	runner := &mockAgentRunner{output: "sync result"}
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "sync task", Async: false})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "sync result" {
		t.Errorf("result = %q", result)
	}
	if tracker.RunningCount() != 0 {
		t.Error("tracker should have no tasks for sync execution")
	}
}

func TestDelegateTask_BuildPromptSimple(t *testing.T) {
	p := buildPrompt("research Go", nil, "")
	if p != "research Go" {
		t.Errorf("prompt = %q", p)
	}
}

func TestDelegateTask_BuildPromptWithSteps(t *testing.T) {
	p := buildPrompt("research Go", []string{"find features", "summarize"}, "")
	if !strings.Contains(p, "1. find features") {
		t.Errorf("prompt missing step 1: %q", p)
	}
	if !strings.Contains(p, "2. summarize") {
		t.Errorf("prompt missing step 2: %q", p)
	}
}

func TestDelegateTask_BuildPromptWithOutputFormat(t *testing.T) {
	p := buildPrompt("research Go", nil, "markdown")
	if !strings.Contains(p, "Output format: markdown") {
		t.Errorf("prompt missing output format: %q", p)
	}
}

func TestDelegateTask_BuildPromptFull(t *testing.T) {
	p := buildPrompt("research Go", []string{"step A"}, "json")
	if !strings.Contains(p, "research Go") {
		t.Error("missing task")
	}
	if !strings.Contains(p, "1. step A") {
		t.Error("missing step")
	}
	if !strings.Contains(p, "Output format: json") {
		t.Error("missing output format")
	}
}

func TestDelegateTask_MaxBudget(t *testing.T) {
	r := NewAgentRunner(AgentInfo{Kind: AgentClaude, Binary: "claude"})
	args := r.buildArgs(runOptions{maxBudgetUSD: 0.50})
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

// mockStreamRunner is a test double for StreamRunnerInterface.
type mockStreamRunner struct {
	output string
	err    error
	events []StreamEvent
}

func (m *mockStreamRunner) RunStream(_ context.Context, _ string, _ time.Duration, onEvent func(StreamEvent), _ ...RunOption) (string, error) {
	for _, evt := range m.events {
		if onEvent != nil {
			onEvent(evt)
		}
	}
	if m.err != nil {
		return "", m.err
	}
	return m.output, nil
}

func TestDelegateTask_AsyncStreamingCompleted(t *testing.T) {
	inter := newSyncMockInteractor(true)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.streamRunners[AgentClaude] = &mockStreamRunner{
		output: "streamed result",
		events: []StreamEvent{
			{Type: "text", Content: "partial"},
			{Type: "result", Content: "streamed result"},
		},
	}

	input, _ := json.Marshal(delegateTaskInput{Task: "stream task", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var resp struct{ TaskID string `json:"task_id"` }
	json.Unmarshal([]byte(result), &resp)

	// Wait for completion notification.
	select {
	case <-inter.notifyCh:
		// May get progress or completion; drain until completion.
		for {
			notified := inter.getNotified()
			for _, n := range notified {
				if strings.Contains(n, "Task completed") {
					goto done
				}
			}
			select {
			case <-inter.notifyCh:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for completion")
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
done:
	rec, ok := tracker.Get(resp.TaskID)
	if !ok {
		t.Fatal("task not found")
	}
	if rec.Status != TaskCompleted {
		t.Errorf("status = %q", rec.Status)
	}
	if rec.Output != "streamed result" {
		t.Errorf("output = %q", rec.Output)
	}
}

func TestDelegateTask_AsyncStreamingFailed(t *testing.T) {
	inter := newSyncMockInteractor(true)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.streamRunners[AgentClaude] = &mockStreamRunner{
		err: errors.New("stream crashed"),
	}

	input, _ := json.Marshal(delegateTaskInput{Task: "fail stream", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var resp struct{ TaskID string `json:"task_id"` }
	json.Unmarshal([]byte(result), &resp)

	select {
	case <-inter.notifyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}

	rec, ok := tracker.Get(resp.TaskID)
	if !ok {
		t.Fatal("task not found")
	}
	if rec.Status != TaskFailed {
		t.Errorf("status = %q", rec.Status)
	}
}

func TestDelegateTask_ProgressThrottling(t *testing.T) {
	inter := newSyncMockInteractor(true)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	// Create many rapid events — only one progress notification should fire
	// (the first one, since all events happen within a single instant).
	events := make([]StreamEvent, 50)
	for i := range events {
		events[i] = StreamEvent{Type: "text", Content: fmt.Sprintf("chunk %d", i)}
	}
	tool.streamRunners[AgentClaude] = &mockStreamRunner{
		output: "final",
		events: events,
	}

	input, _ := json.Marshal(delegateTaskInput{Task: "throttle test", Async: true})
	tool.Execute(context.Background(), input)

	// Wait for completion.
	select {
	case <-inter.notifyCh:
		// Drain all notifications.
		<-time.After(200 * time.Millisecond)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	notified := inter.getNotified()
	progressCount := 0
	for _, n := range notified {
		if strings.Contains(n, "progress") {
			progressCount++
		}
	}
	// With 50 rapid events and 30s throttle, only 1 progress notification should fire.
	if progressCount != 1 {
		t.Errorf("expected 1 progress notification, got %d: %v", progressCount, notified)
	}
}

func TestDelegateTask_ProgressIgnoresToolUseEvents(t *testing.T) {
	inter := newSyncMockInteractor(true)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.streamRunners[AgentClaude] = &mockStreamRunner{
		output: "done",
		events: []StreamEvent{
			{Type: "tool_use", Content: "bash"},
			{Type: "tool_use", Content: "file_read"},
		},
	}

	input, _ := json.Marshal(delegateTaskInput{Task: "tool_use only", Async: true})
	tool.Execute(context.Background(), input)

	select {
	case <-inter.notifyCh:
		<-time.After(200 * time.Millisecond)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}

	notified := inter.getNotified()
	for _, n := range notified {
		if strings.Contains(n, "progress") {
			t.Errorf("tool_use events should not trigger progress: %v", notified)
			break
		}
	}
}

func TestDelegateTask_NoAgentsAvailable(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     nil,
		Timeout:    5 * time.Second,
	})

	input, _ := json.Marshal(delegateTaskInput{Task: "should fail"})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error with no agents")
	}
	if !strings.Contains(err.Error(), "no agents available") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestDelegateTask_AsyncTrackerStartError(t *testing.T) {
	inter := newSyncMockInteractor(true)
	tracker := NewTaskTracker()
	// Fill tracker to max.
	tracker.Start("t1", "task1", AgentClaude)
	tracker.Start("t2", "task2", AgentClaude)
	tracker.Start("t3", "task3", AgentClaude)
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "over limit", Async: true})
	_, err := tool.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when tracker is full")
	}
	// No approval should have been requested.
	inter.mu.Lock()
	approvalCount := len(inter.approvals)
	inter.mu.Unlock()
	if approvalCount != 0 {
		t.Errorf("expected no approvals, got %d", approvalCount)
	}
}

func TestDelegateTask_AsyncDenied(t *testing.T) {
	inter := newSyncMockInteractor(false)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
	})
	runner := &mockAgentRunner{output: "should not run"}
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "denied async", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "denied_by_user" {
		t.Errorf("result = %q", result)
	}
	if runner.called {
		t.Error("runner should not have been called")
	}
	// Task should be marked as failed in tracker.
	tasks := tracker.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in tracker, got %d", len(tasks))
	}
	if tasks[0].Status != TaskFailed {
		t.Errorf("status = %q, want failed", tasks[0].Status)
	}
}

func TestDelegateTask_MalformedJSON(t *testing.T) {
	tool, _, _ := setupDelegateTest(t, true)
	_, err := tool.Execute(context.Background(), json.RawMessage(`{bad json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDelegateTask_WritesResultFile(t *testing.T) {
	scratchDir := t.TempDir()
	inter := &mockInteractor{approveAll: true}
	longOutput := strings.Repeat("line of research output\n", 100)
	runner := &mockAgentRunner{output: longOutput}
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		ScratchDir: scratchDir,
	})
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "research energy crisis"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(result, "Full results") {
		t.Errorf("result should contain file reference: %q", result)
	}
	if !strings.Contains(result, scratchDir) {
		t.Errorf("result should contain scratch dir path: %q", result)
	}
	if strings.Contains(result, strings.Repeat("line of research output\n", 50)) {
		t.Error("full output should not be inline")
	}

	// Verify file was written.
	files, _ := filepath.Glob(filepath.Join(scratchDir, "delegate_*.md"))
	if len(files) != 1 {
		t.Fatalf("expected 1 result file, got %d", len(files))
	}
	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read result file: %v", err)
	}
	if string(content) != longOutput {
		t.Errorf("file content mismatch: got %d bytes, want %d", len(content), len(longOutput))
	}
}

func TestDelegateTask_NoScratchDir_FallsBack(t *testing.T) {
	tool, _, _ := setupDelegateTest(t, true) // no scratch dir
	input, _ := json.Marshal(delegateTaskInput{Task: "research Go 1.23"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "research result" {
		t.Errorf("expected inline result, got %q", result)
	}
}

func TestDelegateTask_EmptyOutput_ReturnsMessage(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	runner := &mockAgentRunner{output: ""}
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		ScratchDir: t.TempDir(),
	})
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "empty research"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "Task produced no output." {
		t.Errorf("result = %q, want %q", result, "Task produced no output.")
	}
}

func TestDelegateTask_AsyncWritesResultFile(t *testing.T) {
	scratchDir := t.TempDir()
	inter := newSyncMockInteractor(true)
	longOutput := strings.Repeat("async research line\n", 100)
	tracker := NewTaskTracker()
	agents := []AgentInfo{{Kind: AgentClaude, Binary: "claude"}}
	tool := NewDelegateTaskTool(DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Timeout:    5 * time.Second,
		Tracker:    tracker,
		ScratchDir: scratchDir,
	})
	runner := newBlockingRunner(longOutput, nil)
	tool.runners[AgentClaude] = runner
	tool.streamRunners[AgentClaude] = nil

	input, _ := json.Marshal(delegateTaskInput{Task: "async research", Async: true})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var resp struct{ TaskID string `json:"task_id"` }
	json.Unmarshal([]byte(result), &resp)

	<-runner.called
	close(runner.release)

	select {
	case <-inter.notifyCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}

	rec, ok := tracker.Get(resp.TaskID)
	if !ok {
		t.Fatal("task not found in tracker")
	}
	if !strings.Contains(rec.Output, "Full results") {
		t.Errorf("tracker should store summary with file path, got: %q", rec.Output)
	}
	if strings.Contains(rec.Output, strings.Repeat("async research line\n", 50)) {
		t.Error("tracker should not store full output")
	}

	files, _ := filepath.Glob(filepath.Join(scratchDir, "delegate_*.md"))
	if len(files) != 1 {
		t.Fatalf("expected 1 result file, got %d", len(files))
	}
	content, _ := os.ReadFile(files[0])
	if string(content) != longOutput {
		t.Errorf("file content mismatch: got %d bytes, want %d", len(content), len(longOutput))
	}
}
