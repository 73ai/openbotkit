package tools

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// mockAgentRunner is a test double for AgentRunnerInterface.
type mockAgentRunner struct {
	output  string
	err     error
	called  bool
	prompt  string
	timeout time.Duration
}

func (m *mockAgentRunner) Run(_ context.Context, prompt string, timeout time.Duration, _ ...RunOption) (string, error) {
	m.called = true
	m.prompt = prompt
	m.timeout = timeout
	if m.err != nil {
		return "", m.err
	}
	return m.output, nil
}

// blockingAgentRunner blocks until released or context is cancelled.
type blockingAgentRunner struct {
	output  string
	err     error
	release chan struct{}
	called  chan struct{}
}

func newBlockingRunner(output string, err error) *blockingAgentRunner {
	return &blockingAgentRunner{
		output:  output,
		err:     err,
		release: make(chan struct{}),
		called:  make(chan struct{}),
	}
}

func (b *blockingAgentRunner) Run(ctx context.Context, _ string, _ time.Duration, _ ...RunOption) (string, error) {
	close(b.called)
	select {
	case <-b.release:
		if b.err != nil {
			return "", b.err
		}
		return b.output, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestDetectAgents_Priority(t *testing.T) {
	agents := DetectAgents()
	if len(agents) == 0 {
		t.Skip("no AI CLIs found on PATH")
	}
	// Verify priority: claude < gemini < codex by index.
	order := map[AgentKind]int{AgentClaude: 0, AgentGemini: 1, AgentCodex: 2}
	prev := -1
	for _, a := range agents {
		idx, ok := order[a.Kind]
		if !ok {
			t.Errorf("unexpected agent kind: %s", a.Kind)
			continue
		}
		if idx <= prev {
			t.Errorf("agent %s (idx %d) came after idx %d — wrong priority", a.Kind, idx, prev)
		}
		prev = idx
		if a.Binary == "" {
			t.Errorf("agent %s has empty binary path", a.Kind)
		}
	}
}

func TestAgentRunner_BuildsClaudeArgs(t *testing.T) {
	r := NewAgentRunner(AgentInfo{Kind: AgentClaude, Binary: "/usr/local/bin/claude"})
	args := r.buildArgs(runOptions{})
	want := []string{"--print", "--output-format", "text"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, want[i])
		}
	}
}

func TestAgentRunner_BuildsGeminiArgs(t *testing.T) {
	r := NewAgentRunner(AgentInfo{Kind: AgentGemini, Binary: "/usr/local/bin/gemini"})
	args := r.buildArgs(runOptions{})
	want := []string{"-p"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	if args[0] != "-p" {
		t.Errorf("args[0] = %q, want %q", args[0], "-p")
	}
}

func TestAgentRunner_BuildsCodexArgs(t *testing.T) {
	r := NewAgentRunner(AgentInfo{Kind: AgentCodex, Binary: "/usr/local/bin/codex"})
	args := r.buildArgs(runOptions{})
	want := []string{"exec"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	if args[0] != "exec" {
		t.Errorf("args[0] = %q, want %q", args[0], "exec")
	}
}

func TestAgentRunner_StripsCLAUDECODE(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	r := NewAgentRunner(AgentInfo{Kind: AgentClaude, Binary: "/usr/local/bin/claude"})
	env := r.buildEnv()
	for _, e := range env {
		if e == "CLAUDECODE=1" {
			t.Error("CLAUDECODE should be stripped from child env")
		}
	}
}

func TestAgentRunner_GeminiKeepsCLAUDECODE(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	r := NewAgentRunner(AgentInfo{Kind: AgentGemini, Binary: "/usr/local/bin/gemini"})
	env := r.buildEnv()
	found := false
	for _, e := range env {
		if e == "CLAUDECODE=1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("CLAUDECODE should NOT be stripped for gemini")
	}
}

func TestAgentRunner_Timeout(t *testing.T) {
	r := NewAgentRunner(AgentInfo{Kind: AgentClaude, Binary: "sleep"})
	_, err := r.Run(context.Background(), "", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestFilterEnv(t *testing.T) {
	env := []string{"HOME=/home/user", "CLAUDECODE=1", "PATH=/usr/bin"}
	got := filterEnv(env, "CLAUDECODE")
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	for _, e := range got {
		if e == "CLAUDECODE=1" {
			t.Error("CLAUDECODE not filtered")
		}
	}
}

func TestAgentRunner_RealClaude(t *testing.T) {
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
	r := NewAgentRunner(info)
	out, err := r.Run(context.Background(), "Say hello in exactly one word.", 30*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestAgentRunner_RealGemini(t *testing.T) {
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
	r := NewAgentRunner(info)
	out, err := r.Run(context.Background(), "Say hello in exactly one word.", 30*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "Permission") || strings.Contains(err.Error(), "denied") || strings.Contains(err.Error(), "auth") {
			t.Skipf("gemini auth not configured: %v", err)
		}
		t.Fatalf("Run: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}
