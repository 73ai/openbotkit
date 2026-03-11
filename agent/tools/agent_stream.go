package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// StreamEvent represents a parsed event from the agent's streaming output.
type StreamEvent struct {
	Type    string // "text", "tool_use", "result", "error"
	Content string
}

// StreamRunnerInterface abstracts streaming agent execution for testability.
type StreamRunnerInterface interface {
	RunStream(ctx context.Context, prompt string, timeout time.Duration, onEvent func(StreamEvent), opts ...RunOption) (string, error)
}

// StreamRunner executes an external AI CLI with streaming NDJSON output.
type StreamRunner struct {
	info AgentInfo
}

// NewStreamRunner creates a streaming runner for the given agent.
func NewStreamRunner(info AgentInfo) *StreamRunner {
	return &StreamRunner{info: info}
}

// RunStream executes the CLI with streaming output, calling onEvent for each
// parsed event. Returns the accumulated text output.
func (r *StreamRunner) RunStream(
	ctx context.Context,
	prompt string,
	timeout time.Duration,
	onEvent func(StreamEvent),
	opts ...RunOption,
) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var ro runOptions
	for _, o := range opts {
		o(&ro)
	}
	args := r.buildStreamArgs(ro)
	if r.info.Kind == AgentGemini {
		args = append(args, "-p", prompt)
	}
	cmd := exec.CommandContext(ctx, r.info.Binary, args...)
	cmd.Env = r.buildEnv()
	if r.info.Kind != AgentGemini {
		cmd.Stdin = strings.NewReader(prompt)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start %s: %w", r.info.Kind, err)
	}

	var accumulated strings.Builder
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		evt := parseStreamLine(line)
		if evt.Type != "" {
			if onEvent != nil {
				onEvent(evt)
			}
			if evt.Type == "result" || evt.Type == "text" {
				accumulated.WriteString(evt.Content)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("agent %s timed out after %s", r.info.Kind, timeout)
		}
		errOutput := stderr.String()
		if errOutput != "" {
			return "", fmt.Errorf("agent %s: %s", r.info.Kind, errOutput)
		}
		return "", fmt.Errorf("agent %s: %w", r.info.Kind, err)
	}

	return accumulated.String(), nil
}

func (r *StreamRunner) buildStreamArgs(opts runOptions) []string {
	switch r.info.Kind {
	case AgentClaude:
		args := []string{"--print", "--verbose", "--output-format", "stream-json"}
		if opts.maxBudgetUSD > 0 {
			args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.maxBudgetUSD))
		}
		return args
	case AgentGemini:
		return []string{"-o", "stream-json"}
	default:
		return nil
	}
}

func (r *StreamRunner) buildEnv() []string {
	env := os.Environ()
	if r.info.Kind == AgentClaude {
		return filterEnv(env, "CLAUDECODE")
	}
	return env
}

// streamJSON is the minimal structure of a Claude stream-json event.
type streamJSON struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Result  string `json:"result"`
}

func parseStreamLine(line []byte) StreamEvent {
	var sj streamJSON
	if err := json.Unmarshal(line, &sj); err != nil {
		return StreamEvent{}
	}
	content := sj.Content
	if content == "" {
		content = sj.Result
	}
	typ := sj.Type
	if typ == "" {
		return StreamEvent{}
	}
	return StreamEvent{Type: typ, Content: content}
}
