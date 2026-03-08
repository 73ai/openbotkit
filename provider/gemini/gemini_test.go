package gemini

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
		if r.URL.Query().Get("key") != "test-key" {
			t.Error("missing or wrong API key")
		}
		json.NewEncoder(w).Encode(apiResponse{
			Candidates: []apiCandidate{{
				Content:      apiContent{Role: "model", Parts: []apiPart{{Text: "Hello!"}}},
				FinishReason: "STOP",
			}},
			UsageMetadata: apiUsage{PromptTokenCount: 10, CandidatesTokenCount: 5},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gemini-2.0-flash",
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
			Candidates: []apiCandidate{{
				Content: apiContent{
					Role: "model",
					Parts: []apiPart{{
						FunctionCall: &apiFuncCall{
							Name: "bash",
							Args: map[string]any{"command": "echo hi"},
						},
					}},
				},
				FinishReason: "STOP",
			}},
		})
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gemini-2.0-flash",
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
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse{
			Error: &apiError{Code: 400, Message: "Invalid API key"},
		})
	}))
	defer server.Close()

	p := New("bad-key", WithBaseURL(server.URL))
	_, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:    "gemini-2.0-flash",
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

		chunks := []apiResponse{
			{Candidates: []apiCandidate{{Content: apiContent{Role: "model", Parts: []apiPart{{Text: "Hello"}}}}}},
			{Candidates: []apiCandidate{{Content: apiContent{Role: "model", Parts: []apiPart{{Text: " world"}}}}}},
			{Candidates: []apiCandidate{{Content: apiContent{Role: "model", Parts: []apiPart{{Text: ""}}}, FinishReason: "STOP"}}},
		}
		for _, c := range chunks {
			data, _ := json.Marshal(c)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}))
	defer server.Close()

	p := New("test-key", WithBaseURL(server.URL))
	ch, err := p.StreamChat(context.Background(), provider.ChatRequest{
		Model:    "gemini-2.0-flash",
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

func TestGeminiIntegration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	p := New(apiKey)
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:     "gemini-2.0-flash",
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

func TestGeminiVertexAIIntegration(t *testing.T) {
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		t.Skip("GOOGLE_CLOUD_PROJECT not set")
	}
	region := os.Getenv("GOOGLE_CLOUD_REGION")
	if region == "" {
		region = "us-east5"
	}
	account := os.Getenv("GOOGLE_CLOUD_ACCOUNT")

	p := New("", WithVertexAI(project, region), WithTokenSource(provider.GcloudTokenSource(account)))
	resp, err := p.Chat(context.Background(), provider.ChatRequest{
		Model:     "gemini-2.0-flash",
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
