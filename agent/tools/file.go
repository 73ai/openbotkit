package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// FileReadTool reads file contents.
type FileReadTool struct{}

func (f *FileReadTool) Name() string        { return "file_read" }
func (f *FileReadTool) Description() string { return "Read the contents of a file" }
func (f *FileReadTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to read"}
		},
		"required": ["path"]
	}`)
}

func (f *FileReadTool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	content, err := os.ReadFile(in.Path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	result := TruncateHead(string(content), MaxLinesFileRead)
	return TruncateBytes(result, MaxOutputBytes), nil
}

// FileWriteTool writes content to a file.
type FileWriteTool struct {
	interactor    Interactor
	approvalRules *ApprovalRuleSet
}

// NewFileWriteTool creates a FileWriteTool with optional approval gate.
func NewFileWriteTool(inter Interactor, rules *ApprovalRuleSet) *FileWriteTool {
	return &FileWriteTool{interactor: inter, approvalRules: rules}
}

func (f *FileWriteTool) Name() string { return "file_write" }
func (f *FileWriteTool) Description() string {
	return "Write content to a file (creates or overwrites)"
}
func (f *FileWriteTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to write"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["path", "content"]
	}`)
}

func (f *FileWriteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	doWrite := func() (string, error) {
		if err := os.WriteFile(in.Path, []byte(in.Content), 0644); err != nil {
			return "", fmt.Errorf("write file: %w", err)
		}
		return fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.Path), nil
	}
	if f.interactor != nil {
		desc := fmt.Sprintf("Write %d bytes to %s", len(in.Content), in.Path)
		return GuardedAction(ctx, f.interactor, RiskMedium, desc, doWrite,
			WithApprovalRules(f.approvalRules, "file_write", input))
	}
	return doWrite()
}

// FileEditTool performs string replacement in a file.
type FileEditTool struct {
	interactor    Interactor
	approvalRules *ApprovalRuleSet
}

// NewFileEditTool creates a FileEditTool with optional approval gate.
func NewFileEditTool(inter Interactor, rules *ApprovalRuleSet) *FileEditTool {
	return &FileEditTool{interactor: inter, approvalRules: rules}
}

func (f *FileEditTool) Name() string        { return "file_edit" }
func (f *FileEditTool) Description() string { return "Replace a string in a file" }
func (f *FileEditTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to edit"},
			"old_string": {"type": "string", "description": "The text to find and replace"},
			"new_string": {"type": "string", "description": "The replacement text"}
		},
		"required": ["path", "old_string", "new_string"]
	}`)
}

func (f *FileEditTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	doEdit := func() (string, error) {
		content, err := os.ReadFile(in.Path)
		if err != nil {
			return "", fmt.Errorf("read file: %w", err)
		}
		original := string(content)
		count := strings.Count(original, in.OldString)
		if count == 0 {
			return "", fmt.Errorf("old_string not found in %s", in.Path)
		}
		updated := strings.Replace(original, in.OldString, in.NewString, 1)
		if err := os.WriteFile(in.Path, []byte(updated), 0644); err != nil {
			return "", fmt.Errorf("write file: %w", err)
		}
		return fmt.Sprintf("replaced 1 occurrence in %s", in.Path), nil
	}

	if f.interactor != nil {
		desc := fmt.Sprintf("Edit %s", in.Path)
		return GuardedAction(ctx, f.interactor, RiskMedium, desc, doEdit,
			WithApprovalRules(f.approvalRules, "file_edit", input))
	}
	return doEdit()
}
