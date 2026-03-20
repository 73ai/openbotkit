package zai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/provider"
)

func TestChat_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
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
		Model:    "glm-4.5-flash",
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
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d", resp.Usage.OutputTokens)
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
		Model:    "glm-4.5-flash",
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
		Model:    "glm-4.5-flash",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChat_ErrorResponseZAICodeFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"code":"1302","message":"Rate limit reached for requests"}}`))
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "glm-4.5-flash",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Hi")},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "1302") || !strings.Contains(got, "Rate limit") {
		t.Errorf("error = %q, want Z.AI code format with 1302", got)
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
		Model:    "glm-4.5-flash",
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

func TestStreamChat_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		chunks := []string{
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"","function":{"name":"","arguments":"{\"cmd\":"}}]},"finish_reason":null}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"","function":{"name":"","arguments":"\"ls\"}"}}]},"finish_reason":null}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "glm-4.5-flash",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "list files")},
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}

	var gotStart, gotDelta, gotEnd bool
	for event := range ch {
		switch event.Type {
		case provider.EventToolCallStart:
			gotStart = true
			if event.ToolCall.Name != "bash" {
				t.Errorf("tool name = %q", event.ToolCall.Name)
			}
		case provider.EventToolCallDelta:
			gotDelta = true
		case provider.EventToolCallEnd:
			gotEnd = true
		}
	}
	if !gotStart {
		t.Error("missing tool_call_start event")
	}
	if !gotDelta {
		t.Error("missing tool_call_delta event")
	}
	if !gotEnd {
		t.Error("missing tool_call_end event")
	}
}

func TestChat_ReasoningContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"","reasoning_content":"Let me think..."},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":10}}`))
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "glm-4.5-flash",
		Messages: []provider.Message{provider.NewTextMessage(provider.RoleUser, "Think about this")},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if text := resp.TextContent(); text != "Let me think..." {
		t.Errorf("text = %q, want reasoning_content fallback", text)
	}
}

func TestChat_RequestFormat(t *testing.T) {
	var capturedBody map[string]any
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
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
		Model:  "glm-4.5-flash",
		System: "You are helpful.",
		Messages: []provider.Message{
			provider.NewTextMessage(provider.RoleUser, "Hi"),
		},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if capturedPath != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", capturedPath)
	}
	if capturedBody["model"] != "glm-4.5-flash" {
		t.Errorf("model = %v", capturedBody["model"])
	}
	if capturedBody["max_tokens"] != float64(100) {
		t.Errorf("max_tokens = %v", capturedBody["max_tokens"])
	}
	msgs := capturedBody["messages"].([]any)
	sysMsg := msgs[0].(map[string]any)
	if sysMsg["role"] != "system" {
		t.Errorf("first message role = %q", sysMsg["role"])
	}
	if sysMsg["content"] != "You are helpful." {
		t.Errorf("system content = %q", sysMsg["content"])
	}
}

func TestZAIIntegration(t *testing.T) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		t.Skip("ZAI_API_KEY not set")
	}

	models := []string{"glm-4.5-flash"}
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			p := New(apiKey)
			resp, err := p.Chat(context.Background(), provider.ChatRequest{
				Model:     model,
				Messages:  []provider.Message{provider.NewTextMessage(provider.RoleUser, "Say 'hello' and nothing else.")},
				MaxTokens: 256,
			})
			if err != nil {
				t.Fatalf("Chat: %v", err)
			}
			t.Logf("StopReason: %q", resp.StopReason)
			if text := resp.TextContent(); text == "" {
				t.Error("empty response")
			} else {
				t.Logf("Response: %q", text)
			}
			t.Logf("Usage: input=%d output=%d", resp.Usage.InputTokens, resp.Usage.OutputTokens)
		})
	}
}
