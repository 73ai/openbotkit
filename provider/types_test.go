package provider

import (
	"encoding/json"
	"testing"
)

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage(RoleUser, "hello")
	if msg.Role != RoleUser {
		t.Errorf("Role = %q", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content length = %d", len(msg.Content))
	}
	if msg.Content[0].Type != ContentText {
		t.Errorf("Type = %q", msg.Content[0].Type)
	}
	if msg.Content[0].Text != "hello" {
		t.Errorf("Text = %q", msg.Content[0].Text)
	}
}

func TestTextContent_Empty(t *testing.T) {
	resp := &ChatResponse{}
	if text := resp.TextContent(); text != "" {
		t.Errorf("TextContent = %q, want empty", text)
	}
}

func TestTextContent_MultipleBlocks(t *testing.T) {
	resp := &ChatResponse{
		Content: []ContentBlock{
			{Type: ContentText, Text: "hello "},
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "1", Name: "bash"}},
			{Type: ContentText, Text: "world"},
		},
	}
	if text := resp.TextContent(); text != "hello world" {
		t.Errorf("TextContent = %q, want %q", text, "hello world")
	}
}

func TestToolCalls_Empty(t *testing.T) {
	resp := &ChatResponse{
		Content: []ContentBlock{{Type: ContentText, Text: "no tools"}},
	}
	if calls := resp.ToolCalls(); len(calls) != 0 {
		t.Errorf("ToolCalls length = %d, want 0", len(calls))
	}
}

func TestToolCalls_Multiple(t *testing.T) {
	resp := &ChatResponse{
		Content: []ContentBlock{
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}},
			{Type: ContentText, Text: "between"},
			{Type: ContentToolUse, ToolCall: &ToolCall{ID: "2", Name: "file_read", Input: json.RawMessage(`{}`)}},
		},
	}
	calls := resp.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("ToolCalls length = %d, want 2", len(calls))
	}
	if calls[0].Name != "bash" || calls[1].Name != "file_read" {
		t.Errorf("names = %q, %q", calls[0].Name, calls[1].Name)
	}
}
