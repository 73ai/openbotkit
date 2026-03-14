package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultBashTimeout = 30 * time.Second

// BashTool executes shell commands.
type BashTool struct {
	timeout time.Duration
	filter  *CommandFilter
	workDir string
}

// BashOption configures a BashTool.
type BashOption func(*BashTool)

// WithCommandFilter sets a command filter on the bash tool.
func WithCommandFilter(f *CommandFilter) BashOption {
	return func(b *BashTool) { b.filter = f }
}

// WithWorkDir sets the working directory for command execution.
func WithWorkDir(dir string) BashOption {
	return func(b *BashTool) { b.workDir = dir }
}

// NewBashTool creates a new bash tool with the given timeout and options.
func NewBashTool(timeout time.Duration, opts ...BashOption) *BashTool {
	if timeout == 0 {
		timeout = defaultBashTimeout
	}
	b := &BashTool{timeout: timeout}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *BashTool) Name() string        { return "bash" }
func (b *BashTool) Description() string { return "Execute a shell command and return its output" }
func (b *BashTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute"
			}
		},
		"required": ["command"]
	}`)
}

type bashInput struct {
	Command string `json:"command"`
}

func (b *BashTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in bashInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	if strings.HasPrefix(strings.TrimSpace(in.Command), "gws ") {
		return "", fmt.Errorf("gws commands must use the gws_execute tool, not bash")
	}

	if err := b.filter.Check(in.Command); err != nil {
		return "", fmt.Errorf("command blocked: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", in.Command)
	if b.workDir != "" {
		cmd.Dir = b.workDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := stdout.String()
	if stderr.Len() > 0 {
		result += "\nSTDERR:\n" + stderr.String()
	}

	result = TruncateTail(result, MaxLinesBash)
	result = TruncateBytes(result, MaxOutputBytes)

	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("command timed out after %s", b.timeout)
	}
	if err != nil {
		return result, fmt.Errorf("command failed: %w", err)
	}

	return result, nil
}
