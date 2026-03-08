package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
)

func TestChat_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing or wrong API key")
		}
		if r.Header.Get("anthropic-version") != apiVersion {
			t.Error("missing or wrong anthropic-version")
		}
		json.NewEncoder(w).Encode(apiResponse{
			Type: "message",
			Content: []apiContent{
				{Type: "text", Text: "Hello! How can I help?"},
			},
			StopReason: "end_turn",
			Usage:      apiUsage{InputTokens: 10, OutputTokens: 8},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hello")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if resp.StopReason != provider.StopEndTurn {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, provider.StopEndTurn)
	}
	if text := resp.TextContent(); text != "Hello! How can I help?" {
		t.Errorf("TextContent = %q", text)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
}

func TestChat_ToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Type: "message",
			Content: []apiContent{
				{Type: "text", Text: "Let me check that."},
				{
					Type:  "tool_use",
					ID:    "call_123",
					Name:  "bash",
					Input: json.RawMessage(`{"command":"echo hello"}`),
				},
			},
			StopReason: "tool_use",
			Usage:      apiUsage{InputTokens: 20, OutputTokens: 15},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "run echo hello")},
		Tools: []provider.Tool{
			{Name: "bash", Description: "Run a command", InputSchema: json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"}}}`)},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if resp.StopReason != provider.StopToolUse {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, provider.StopToolUse)
	}

	calls := resp.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(calls))
	}
	if calls[0].ID != "call_123" {
		t.Errorf("ToolCall.ID = %q", calls[0].ID)
	}
	if calls[0].Name != "bash" {
		t.Errorf("ToolCall.Name = %q", calls[0].Name)
	}
}

func TestChat_MultipleToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Type: "message",
			Content: []apiContent{
				{
					Type:  "tool_use",
					ID:    "call_1",
					Name:  "file_read",
					Input: json.RawMessage(`{"path":"/tmp/a.txt"}`),
				},
				{
					Type:  "tool_use",
					ID:    "call_2",
					Name:  "file_read",
					Input: json.RawMessage(`{"path":"/tmp/b.txt"}`),
				},
			},
			StopReason: "tool_use",
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "read both files")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	calls := resp.ToolCalls()
	if len(calls) != 2 {
		t.Fatalf("got %d tool calls, want 2", len(calls))
	}
	if calls[0].ID != "call_1" || calls[1].ID != "call_2" {
		t.Errorf("unexpected call IDs: %q, %q", calls[0].ID, calls[1].ID)
	}
}

func TestChat_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(apiResponse{
			Type:  "error",
			Error: apiError{Type: "rate_limit_error", Message: "Too many requests"},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hello")},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "anthropic API error (HTTP 429): rate_limit_error: Too many requests" {
		t.Errorf("error = %q", got)
	}
}

func TestStreamChat_TextDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		events := []string{
			`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[]}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
		}
		for _, e := range events {
			fmt.Fprintf(w, "data: %s\n\n", e)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hello")},
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var text string
	for event := range ch {
		if event.Type == provider.EventTextDelta {
			text += event.Text
		}
		if event.Type == provider.EventError {
			t.Fatalf("stream error: %v", event.Error)
		}
	}

	if text != "Hello world" {
		t.Errorf("streamed text = %q, want %q", text, "Hello world")
	}
}

func TestStreamChat_ToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		events := []string{
			`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_1","name":"bash"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"command\":"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"ls\"}"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","delta":{"stop_reason":"tool_use"}}`,
		}
		for _, e := range events {
			fmt.Fprintf(w, "data: %s\n\n", e)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "list files")},
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var gotStart, gotEnd bool
	var delta string
	for event := range ch {
		switch event.Type {
		case provider.EventToolCallStart:
			gotStart = true
			if event.ToolCall.Name != "bash" {
				t.Errorf("tool name = %q", event.ToolCall.Name)
			}
		case provider.EventToolCallDelta:
			delta += event.Delta
		case provider.EventToolCallEnd:
			gotEnd = true
		case provider.EventError:
			t.Fatalf("stream error: %v", event.Error)
		}
	}

	if !gotStart || !gotEnd {
		t.Errorf("start=%v end=%v, want both true", gotStart, gotEnd)
	}
	if delta != `{"command":"ls"}` {
		t.Errorf("delta = %q", delta)
	}
}

func TestAnthropicIntegration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	p := New(apiKey)
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:     "claude-sonnet-4-6",
		Messages:  []provider.Message{provider.NewTextMessage(provider.RoleUser, "Say 'hello' and nothing else.")},
		MaxTokens: 32,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if resp.StopReason != provider.StopEndTurn {
		t.Errorf("StopReason = %q", resp.StopReason)
	}
	if text := resp.TextContent(); text == "" {
		t.Error("empty response")
	}
}

func TestVertexAIIntegration(t *testing.T) {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		t.Skip("GOOGLE_CLOUD_PROJECT not set")
	}
	region := os.Getenv("GOOGLE_CLOUD_REGION")
	if region == "" {
		region = "us-east5"
	}
	model := os.Getenv("VERTEX_CLAUDE_MODEL")
	if model == "" {
		model = "claude-sonnet-4@20250514"
	}
	account := os.Getenv("GOOGLE_CLOUD_ACCOUNT")

	p := New("", WithVertexAI(project, region), WithTokenSource(provider.GcloudTokenSource(account)))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:     model,
		Messages:  []provider.Message{provider.NewTextMessage(provider.RoleUser, "Say 'hello' and nothing else.")},
		MaxTokens: 32,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if resp.StopReason != provider.StopEndTurn {
		t.Errorf("StopReason = %q", resp.StopReason)
	}
	if text := resp.TextContent(); text == "" {
		t.Error("empty response")
	}
}
