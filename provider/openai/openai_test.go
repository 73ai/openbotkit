package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/73ai/openbotkit/provider"
)

func TestChat_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong Authorization header")
		}
		json.NewEncoder(w).Encode(apiResponse{
			Choices: []apiChoice{
				{
					Message:      apiMessage{Role: "assistant", Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: apiUsage{PromptTokens: 10, CompletionTokens: 5},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hello")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.StopReason != provider.StopEndTurn {
		t.Errorf("StopReason = %q", resp.StopReason)
	}
	if text := resp.TextContent(); text != "Hello!" {
		t.Errorf("text = %q", text)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d", resp.Usage.InputTokens)
	}
}

func TestChat_ToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Choices: []apiChoice{
				{
					Message: apiMessage{
						Role: "assistant",
						ToolCalls: []apiToolCall{
							{
								ID:   "call_abc",
								Type: "function",
								Function: apiFunction{
									Name:      "bash",
									Arguments: `{"command":"echo hi"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "run echo hi")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.StopReason != provider.StopToolUse {
		t.Errorf("StopReason = %q", resp.StopReason)
	}
	calls := resp.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(calls))
	}
	if calls[0].Name != "bash" {
		t.Errorf("tool name = %q", calls[0].Name)
	}
}

func TestChat_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(apiResponse{
			Error: &apiError{Type: "invalid_api_key", Message: "Invalid API key"},
		})
	}))
	defer server.Close()

	p := New("bad-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStreamChat_TextDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []string{
			`{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"choices":[{"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var text string
	for event := range ch {
		if event.Type == provider.EventTextDelta {
			text += event.Text
		}
	}
	if text != "Hello world" {
		t.Errorf("text = %q", text)
	}
}

func TestChat_CachedTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Choices: []apiChoice{
				{Message: apiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"},
			},
			Usage: apiUsage{
				PromptTokens:     100,
				CompletionTokens: 10,
				PromptTokensDetails: &promptTokensDetails{
					CachedTokens: 80,
				},
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Usage.CacheReadTokens != 80 {
		t.Errorf("CacheReadTokens = %d, want 80", resp.Usage.CacheReadTokens)
	}
	if resp.Usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", resp.Usage.InputTokens)
	}
}

func TestChat_NoCachedTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(apiResponse{
			Choices: []apiChoice{
				{Message: apiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"},
			},
			Usage: apiUsage{PromptTokens: 50, CompletionTokens: 5},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gpt-4o",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Usage.CacheReadTokens != 0 {
		t.Errorf("CacheReadTokens = %d, want 0", resp.Usage.CacheReadTokens)
	}
}

func TestChat_SystemBlocks(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(apiResponse{
			Choices: []apiChoice{
				{Message: apiMessage{Role: "assistant", Content: "ok"}, FinishReason: "stop"},
			},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Model: "gpt-4o",
		SystemBlocks: []provider.SystemBlock{
			{Text: "Part 1. "},
			{Text: "Part 2."},
		},
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	msgs := capturedBody["messages"].([]any)
	sysMsg := msgs[0].(map[string]any)
	if sysMsg["role"] != "system" {
		t.Errorf("first message role = %q", sysMsg["role"])
	}
	if sysMsg["content"] != "Part 1. Part 2." {
		t.Errorf("system content = %q", sysMsg["content"])
	}
}

func TestOpenAIIntegration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	p := New(apiKey)
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:     "gpt-4o-mini",
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
