package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/provider"
)

func TestRegistryProviderTools(t *testing.T) {
	r := NewRegistry()
	r.Register(NewBashTool(0))
	r.Register(&FileReadTool{})

	schemas := r.ToolSchemas()
	if len(schemas) != 2 {
		t.Fatalf("got %d tools, want 2", len(schemas))
	}

	// Verify schemas are valid JSON.
	for _, s := range schemas {
		if !json.Valid(s.InputSchema) {
			t.Errorf("tool %q has invalid JSON schema", s.Name)
		}
	}
}

func TestBashEcho(t *testing.T) {
	b := NewBashTool(5 * time.Second)
	result, err := b.Execute(context.Background(), json.RawMessage(`{"command":"echo hello"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.TrimSpace(result) != "hello" {
		t.Errorf("result = %q, want %q", result, "hello\n")
	}
}

func TestBashTimeout(t *testing.T) {
	b := NewBashTool(1 * time.Second)
	_, err := b.Execute(context.Background(), json.RawMessage(`{"command":"sleep 10"}`))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, expected timeout", err)
	}
}

func TestBashStderr(t *testing.T) {
	b := NewBashTool(5 * time.Second)
	result, _ := b.Execute(context.Background(), json.RawMessage(`{"command":"echo oops >&2"}`))
	if !strings.Contains(result, "oops") {
		t.Errorf("stderr not captured: %q", result)
	}
}

func TestFileRead(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	f := &FileReadTool{}
	input, _ := json.Marshal(map[string]string{"path": path})
	result, err := f.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "hello world" {
		t.Errorf("result = %q", result)
	}
}

func TestFileWrite(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.txt")

	f := &FileWriteTool{}
	input, _ := json.Marshal(map[string]string{"path": path, "content": "new content"})
	_, err := f.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("content = %q", string(got))
	}
}

func TestFileEdit(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "edit.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	f := &FileEditTool{}
	input, _ := json.Marshal(map[string]string{
		"path":       path,
		"old_string": "world",
		"new_string": "there",
	})
	_, err := f.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "hello there" {
		t.Errorf("content = %q", string(got))
	}
}

func TestRegistryTruncatesLargeOutput(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{
		name:   "big",
		output: strings.Repeat("x", 600_000), // ~600KB
	})

	output, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "big",
		Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(output) >= 600_000 {
		t.Errorf("output not truncated: len=%d", len(output))
	}
	if !strings.Contains(output, "output truncated") {
		t.Errorf("missing truncation notice")
	}
}

func TestRegistrySmallOutputUnchanged(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "small", output: "hello"})

	output, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "small",
		Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if output != "hello" {
		t.Errorf("output = %q, want %q", output, "hello")
	}
}

// stubTool is a minimal tool for testing the registry.
type stubTool struct {
	name   string
	output string
}

func (s *stubTool) Name() string                                                     { return s.name }
func (s *stubTool) Description() string                                              { return "stub" }
func (s *stubTool) InputSchema() json.RawMessage                                     { return json.RawMessage(`{"type":"object"}`) }
func (s *stubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return s.output, nil }

func TestRegistryExactlyAtLimit(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "exact", output: strings.Repeat("x", 524288)})

	output, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "exact",
		Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if output != strings.Repeat("x", 524288) {
		t.Errorf("output at exact limit should not be truncated")
	}
}

func TestRegistryOneOverLimit(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "over", output: strings.Repeat("x", 524289)})

	output, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "over",
		Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(output, "output truncated") {
		t.Error("expected truncation for 1 byte over limit")
	}
}

func TestRegistryEmptyOutput(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "empty", output: ""})

	output, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "empty",
		Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if output != "" {
		t.Errorf("output = %q, want empty", output)
	}
}

func TestRegistryUnknownTool(t *testing.T) {
	r := NewRegistry()

	_, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "nonexistent",
		Input: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("error = %q, expected unknown tool error", err.Error())
	}
}

func TestFileEditNotFound(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "edit.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	f := &FileEditTool{}
	input, _ := json.Marshal(map[string]string{
		"path":       path,
		"old_string": "xyz",
		"new_string": "abc",
	})
	_, err := f.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for missing old_string")
	}
}
