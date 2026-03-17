package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/provider"
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

func TestToolSchemasDeterministicOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "zeta"})
	r.Register(&stubTool{name: "alpha"})
	r.Register(&stubTool{name: "middle"})

	var first []string
	for _, s := range r.ToolSchemas() {
		first = append(first, s.Name)
	}

	for range 10 {
		var names []string
		for _, s := range r.ToolSchemas() {
			names = append(names, s.Name)
		}
		if len(names) != len(first) {
			t.Fatalf("length changed: %d vs %d", len(names), len(first))
		}
		for i, name := range names {
			if name != first[i] {
				t.Fatalf("order changed at index %d: %q vs %q", i, name, first[i])
			}
		}
	}

	// Verify alphabetical order.
	if first[0] != "alpha" || first[1] != "middle" || first[2] != "zeta" {
		t.Errorf("expected alphabetical order, got %v", first)
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

func TestBashTool_TruncatesLargeOutput(t *testing.T) {
	b := NewBashTool(10 * time.Second)
	// seq 5000 produces 5000 lines; truncation should keep last 2000.
	result, err := b.Execute(context.Background(), json.RawMessage(`{"command":"seq 5000"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "[truncated: showing 2000 of") {
		t.Error("expected truncation marker for >2000 lines")
	}
	// Last line should be "5000" (tail truncation keeps the end).
	if !strings.HasSuffix(strings.TrimSpace(result), "5000") {
		t.Errorf("expected last line to be 5000, got tail: %q",
			result[max(0, len(result)-50):])
	}
}

func TestBashTool_TruncatesLargeBytes(t *testing.T) {
	b := NewBashTool(10 * time.Second)
	// Generate >50KB of single-line output.
	result, err := b.Execute(context.Background(), json.RawMessage(`{"command":"head -c 60000 /dev/zero | tr '\\0' 'x'"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(result) > 55*1024 { // 50KB + some marker overhead
		t.Errorf("result too large: %d bytes, expected <=~50KB", len(result))
	}
}

func TestBashBlocksGWS(t *testing.T) {
	b := NewBashTool(5 * time.Second)
	_, err := b.Execute(context.Background(), json.RawMessage(`{"command":"gws calendar events.list"}`))
	if err == nil {
		t.Fatal("expected error for gws command in bash")
	}
	if !strings.Contains(err.Error(), "gws_execute") {
		t.Errorf("error = %q, expected gws_execute reference", err.Error())
	}
}

func TestBashAllowsEchoGWS(t *testing.T) {
	b := NewBashTool(5 * time.Second)
	result, err := b.Execute(context.Background(), json.RawMessage(`{"command":"echo gws is great"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result, "gws is great") {
		t.Errorf("result = %q", result)
	}
}

func TestRegistryExactlyAtLimit(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "exact", output: strings.Repeat("x", 102400)})

	output, err := r.Execute(context.Background(), provider.ToolCall{
		Name:  "exact",
		Input: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if output != strings.Repeat("x", 102400) {
		t.Errorf("output at exact limit should not be truncated")
	}
}

func TestRegistryOneOverLimit(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubTool{name: "over", output: strings.Repeat("x", 102401)})

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

func TestBuildBaseSystemPrompt_GWSInstructions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "gws_execute"})
	prompt := BuildBaseSystemPrompt(reg)
	if !strings.Contains(prompt, "Google Workspace") {
		t.Error("prompt missing gws_execute instructions")
	}
	if !strings.Contains(prompt, "gws_execute") {
		t.Error("prompt missing gws_execute tool reference")
	}
	if !strings.Contains(prompt, "load_skills") {
		t.Error("GWS section should instruct agent to load skills before gws_execute")
	}
}

func TestBuildBaseSystemPrompt_NoGWSInstructions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "bash"})
	prompt := BuildBaseSystemPrompt(reg)
	if strings.Contains(prompt, "Google Workspace") {
		t.Error("prompt should not contain gws instructions without gws_execute")
	}
}

func TestBuildBaseSystemPrompt_DelegateInstructions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "delegate_task"})
	prompt := BuildBaseSystemPrompt(reg)
	if !strings.Contains(prompt, "Task Delegation") {
		t.Error("prompt missing delegate_task instructions")
	}
}

func TestBuildBaseSystemPrompt_NoDelegateInstructions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "bash"})
	prompt := BuildBaseSystemPrompt(reg)
	if strings.Contains(prompt, "Task Delegation") {
		t.Error("prompt should not contain delegate instructions without delegate_task")
	}
}

func TestBuildBaseSystemPrompt_SlackInstructions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "slack_search"})
	prompt := BuildBaseSystemPrompt(reg)
	if !strings.Contains(prompt, "Slack") {
		t.Error("prompt missing slack instructions")
	}
}

func TestBuildBaseSystemPrompt_NoSlackInstructions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "bash"})
	prompt := BuildBaseSystemPrompt(reg)
	if strings.Contains(prompt, "slack_search") {
		t.Error("prompt should not contain slack instructions without slack tools")
	}
}

func TestBuildSystemBlocks_BaseOnly(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "bash"})

	blocks := BuildSystemBlocks("You are an AI.\n", reg)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[0].CacheControl == nil || blocks[0].CacheControl.Type != "ephemeral" {
		t.Error("base block should have ephemeral cache_control")
	}
	if !strings.Contains(blocks[0].Text, "You are an AI.") {
		t.Error("base block should contain identity")
	}
	if !strings.Contains(blocks[0].Text, "bash") {
		t.Error("base block should contain tool names")
	}
	if !strings.Contains(blocks[1].Text, "Current date and time:") {
		t.Error("extras block should contain today's date")
	}
}

func TestBuildSystemBlocks_WithExtras(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubTool{name: "bash"})

	blocks := BuildSystemBlocks("You are an AI.\n", reg, "\nBe concise.\n", "User likes Go.")
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[1].CacheControl != nil {
		t.Error("extras block should not have cache_control")
	}
	if !strings.Contains(blocks[1].Text, "Current date and time:") {
		t.Error("extras block should contain today's date")
	}
	if !strings.Contains(blocks[1].Text, "Be concise.") || !strings.Contains(blocks[1].Text, "User likes Go.") {
		t.Errorf("extras block should contain caller extras, got %q", blocks[1].Text)
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
