package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const sandboxTimeout = 30 * time.Second

// SandboxExecTool executes code in a sandboxed environment.
type SandboxExecTool struct {
	runtime SandboxRuntime
}

// NewSandboxExecTool creates a sandbox execution tool.
// If runtime is nil, the tool reports that sandboxing is unavailable.
func NewSandboxExecTool(runtime SandboxRuntime) *SandboxExecTool {
	return &SandboxExecTool{runtime: runtime}
}

func (s *SandboxExecTool) Name() string { return "sandbox_exec" }
func (s *SandboxExecTool) Description() string {
	return "Execute code in a sandboxed environment with read-only filesystem and no network access"
}
func (s *SandboxExecTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"language": {"type": "string", "enum": ["python", "bash", "node", "ruby"], "description": "Language to execute"},
			"code": {"type": "string", "description": "Code to execute"}
		},
		"required": ["language", "code"]
	}`)
}

type sandboxExecInput struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

func (s *SandboxExecTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in sandboxExecInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Code == "" {
		return "", fmt.Errorf("code is required")
	}

	interpreter, ok := LanguageInterpreters[in.Language]
	if !ok {
		return "", fmt.Errorf("unsupported language %q (use python, bash, node, or ruby)", in.Language)
	}

	if s.runtime == nil || !s.runtime.Available() {
		return "sandbox not available on this system (install bwrap on Linux or use macOS); use bash tool instead (requires approval)", nil
	}

	tmpDir, err := os.MkdirTemp("", "obk-sandbox-exec-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	ext := languageExtension(in.Language)
	codePath := filepath.Join(tmpDir, "code"+ext)
	if err := os.WriteFile(codePath, []byte(in.Code), 0600); err != nil {
		return "", fmt.Errorf("write code file: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, sandboxTimeout)
	defer cancel()

	command := interpreter + " " + codePath
	output, err := s.runtime.Exec(ctx, command, tmpDir)

	output = TruncateTail(output, MaxLinesBash)
	output = TruncateBytes(output, MaxOutputBytes)

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("sandbox execution timed out after %s", sandboxTimeout)
	}
	if err != nil {
		return output, fmt.Errorf("sandbox execution failed: %w", err)
	}
	return output, nil
}

func languageExtension(lang string) string {
	switch lang {
	case "python":
		return ".py"
	case "node":
		return ".js"
	case "ruby":
		return ".rb"
	default:
		return ".sh"
	}
}
