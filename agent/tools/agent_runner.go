package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// AgentKind identifies an external AI CLI agent.
type AgentKind string

const (
	AgentClaude AgentKind = "claude"
	AgentGemini AgentKind = "gemini"
	AgentCodex  AgentKind = "codex"
)

// AgentInfo describes a detected external CLI agent.
type AgentInfo struct {
	Kind   AgentKind
	Binary string // absolute path from LookPath
}

// DetectAgents scans PATH for known AI CLI agents in priority order.
func DetectAgents() []AgentInfo {
	candidates := []AgentKind{AgentClaude, AgentGemini, AgentCodex}
	var found []AgentInfo
	for _, kind := range candidates {
		if bin, err := exec.LookPath(string(kind)); err == nil {
			found = append(found, AgentInfo{Kind: kind, Binary: bin})
		}
	}
	return found
}

// AgentRunnerInterface abstracts agent CLI execution for testability.
type AgentRunnerInterface interface {
	Run(ctx context.Context, prompt string, timeout time.Duration) (string, error)
}

// AgentRunner executes an external AI CLI agent.
type AgentRunner struct {
	info AgentInfo
}

// NewAgentRunner creates a runner for the given agent.
func NewAgentRunner(info AgentInfo) *AgentRunner {
	return &AgentRunner{info: info}
}

// Run executes the CLI with the given prompt and timeout, returning stdout.
func (r *AgentRunner) Run(ctx context.Context, prompt string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := r.buildArgs()
	cmd := exec.CommandContext(ctx, r.info.Binary, args...)
	cmd.Env = r.buildEnv()
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("agent %s timed out after %s", r.info.Kind, timeout)
		}
		combined := stdout.String() + stderr.String()
		if combined != "" {
			return "", fmt.Errorf("agent %s: %s", r.info.Kind, combined)
		}
		return "", fmt.Errorf("agent %s: %w", r.info.Kind, err)
	}
	return stdout.String(), nil
}

func (r *AgentRunner) buildArgs() []string {
	switch r.info.Kind {
	case AgentClaude:
		return []string{"--print", "--output-format", "text"}
	case AgentGemini:
		return []string{"-p"}
	default:
		return nil
	}
}

func (r *AgentRunner) buildEnv() []string {
	env := os.Environ()
	if r.info.Kind == AgentClaude {
		return filterEnv(env, "CLAUDECODE")
	}
	return env
}

// filterEnv returns env with entries matching the given key prefix removed.
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
