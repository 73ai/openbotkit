package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CommandRunner abstracts shell execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, args []string, env []string) (string, error)
}

// GWSRunner executes gws commands.
type GWSRunner struct{}

func NewGWSRunner() *GWSRunner { return &GWSRunner{} }

func (r *GWSRunner) Run(ctx context.Context, args []string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, "gws", args...)
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String() + stderr.String(), fmt.Errorf("gws: %w", err)
	}
	return stdout.String(), nil
}
