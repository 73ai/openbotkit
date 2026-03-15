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

// RunOption configures an agent run.
type RunOption func(*runOptions)

type runOptions struct {
	maxBudgetUSD float64
}

// WithMaxBudget sets the maximum API cost budget (Claude only).
func WithMaxBudget(usd float64) RunOption {
	return func(o *runOptions) { o.maxBudgetUSD = usd }
}

// AgentRunnerInterface abstracts agent CLI execution for testability.
type AgentRunnerInterface interface {
	Run(ctx context.Context, prompt string, timeout time.Duration, opts ...RunOption) (string, error)
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
func (r *AgentRunner) Run(ctx context.Context, prompt string, timeout time.Duration, opts ...RunOption) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var ro runOptions
	for _, o := range opts {
		o(&ro)
	}
	args := r.buildArgs(ro)
	// Gemini takes prompt as -p argument; others use stdin.
	if r.info.Kind == AgentGemini {
		args = append(args, prompt)
	}
	cmd := exec.CommandContext(ctx, r.info.Binary, args...)
	cmd.Env = r.buildEnv()
	if r.info.Kind != AgentGemini {
		cmd.Stdin = strings.NewReader(prompt)
	}

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

func (r *AgentRunner) buildArgs(opts runOptions) []string {
	switch r.info.Kind {
	case AgentClaude:
		args := []string{"--print", "--output-format", "text", "--dangerously-skip-permissions"}
		if opts.maxBudgetUSD > 0 {
			args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.maxBudgetUSD))
		}
		return args
	case AgentGemini:
		return []string{"--approval-mode", "yolo", "-p"}
	case AgentCodex:
		return []string{"exec", "--full-auto"}
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
