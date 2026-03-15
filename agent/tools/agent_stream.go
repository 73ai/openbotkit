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
		args := []string{"--print", "--verbose", "--output-format", "stream-json", "--dangerously-skip-permissions"}
		if opts.maxBudgetUSD > 0 {
			args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", opts.maxBudgetUSD))
		}
		return args
	case AgentGemini:
		return []string{"--approval-mode", "yolo", "-o", "stream-json"}
	case AgentCodex:
		return []string{"exec", "--json", "--full-auto"}
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

// streamJSON is a unified structure for parsing streaming events from all CLIs.
type streamJSON struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Result  string `json:"result"`
	Role    string `json:"role"`
	Delta   bool   `json:"delta"`
	Item    struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"item"`
}

func parseStreamLine(line []byte) StreamEvent {
	var sj streamJSON
	if err := json.Unmarshal(line, &sj); err != nil {
		return StreamEvent{}
	}
	if sj.Type == "" {
		return StreamEvent{}
	}

	// Gemini: {"type":"message","role":"assistant","content":"Hello","delta":true}
	if sj.Type == "message" && sj.Role == "assistant" {
		return StreamEvent{Type: "text", Content: sj.Content}
	}

	// Codex: {"type":"item.completed","item":{"type":"agent_message","text":"Hello"}}
	if sj.Type == "item.completed" && sj.Item.Type == "agent_message" {
		return StreamEvent{Type: "text", Content: sj.Item.Text}
	}

	// Drop non-text Gemini/Codex protocol events (user messages, reasoning, etc.)
	if sj.Type == "message" || sj.Type == "item.completed" ||
		sj.Type == "init" || sj.Type == "thread.started" ||
		sj.Type == "turn.started" || sj.Type == "turn.completed" {
		return StreamEvent{}
	}

	// Claude: {"type":"text"|"result"|"tool_use", "content":"...", "result":"..."}
	content := sj.Content
	if content == "" {
		content = sj.Result
	}
	return StreamEvent{Type: sj.Type, Content: content}
}
