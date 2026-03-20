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
	env := scrubEnv(os.Environ())
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

// scrubEnv removes environment variables that contain sensitive values
// (API keys, tokens, secrets, passwords) from the given list.
func scrubEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		eqIdx := strings.IndexByte(e, '=')
		if eqIdx < 0 {
			filtered = append(filtered, e)
			continue
		}
		key := e[:eqIdx]
		if !isSensitiveKey(key) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

var safeKeys = map[string]bool{
	"PATH": true, "HOME": true, "USER": true, "SHELL": true,
	"TERM": true, "LANG": true, "TMPDIR": true, "PWD": true,
	"GOPATH": true, "GOROOT": true, "GOBIN": true,
	"EDITOR": true, "VISUAL": true, "PAGER": true,
	"HOSTNAME": true, "LOGNAME": true, "DISPLAY": true,
	"COLORTERM": true, "TERM_PROGRAM": true, "SHLVL": true,
}

var safePrefixes = []string{"LC_", "XDG_"}

var sensitivePrefixes = []string{
	"ANTHROPIC_", "OPENAI_", "GOOGLE_API_", "GEMINI_",
	"GROQ_", "OPENROUTER_", "AWS_SECRET_", "CLAUDECODE",
}

var sensitiveSuffixes = []string{
	"_KEY", "_SECRET", "_TOKEN", "_PASSWORD", "_CREDENTIAL", "_AUTH",
	"_PRIVATE_KEY", "_DSN",
}

var sensitiveExact = map[string]bool{
	"GITHUB_TOKEN": true, "GH_TOKEN": true,
	"DATABASE_URL": true, "REDIS_URL": true, "MONGODB_URI": true,
	"AMQP_URL": true, "ELASTICSEARCH_URL": true,
}

func isSensitiveKey(key string) bool {
	if safeKeys[key] {
		return false
	}
	for _, p := range safePrefixes {
		if strings.HasPrefix(key, p) {
			return false
		}
	}
	if sensitiveExact[key] {
		return true
	}
	upper := strings.ToUpper(key)
	for _, p := range sensitivePrefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	for _, s := range sensitiveSuffixes {
		if strings.HasSuffix(upper, s) {
			return true
		}
	}
	return false
}
