package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

const defaultBashTimeout = 30 * time.Second

// BashTool executes shell commands.
type BashTool struct {
	timeout time.Duration
}

// NewBashTool creates a new bash tool with the given timeout.
func NewBashTool(timeout time.Duration) *BashTool {
	if timeout == 0 {
		timeout = defaultBashTimeout
	}
	return &BashTool{timeout: timeout}
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

	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", in.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := stdout.String()
	if stderr.Len() > 0 {
		result += "\nSTDERR:\n" + stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("command timed out after %s", b.timeout)
	}
	if err != nil {
		return result, fmt.Errorf("command failed: %w", err)
	}

	return result, nil
}
