package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/source/websearch"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	chatResp *provider.ChatResponse
	chatErr  error
	lastReq  provider.ChatRequest
}

func (m *mockProvider) Chat(_ context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.lastReq = req
	return m.chatResp, m.chatErr
}

func (m *mockProvider) StreamChat(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	return nil, errors.New("not implemented")
}

func TestWebFetchTool_Name(t *testing.T) {
	tool := NewWebFetchTool(WebToolDeps{})
	if tool.Name() != "web_fetch" {
		t.Fatalf("expected name web_fetch, got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("expected non-empty description")
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema(), &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}
}

func TestWebFetchTool_ShortContent(t *testing.T) {
	mock := &mockWebSearcher{
		fetchResult: &websearch.FetchResult{
			URL:     "https://example.com",
			Content: "Hello world, this is short content.",
		},
	}
	mp := &mockProvider{}
	tool := NewWebFetchTool(WebToolDeps{WS: mock, Provider: mp, Model: "test-model"})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","question":"what is this?"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Source: https://example.com") {
		t.Error("expected source URL")
	}
	if !strings.Contains(out, "Hello world") {
		t.Error("expected content returned directly")
	}
	if mp.lastReq.Model != "" {
		t.Error("provider should NOT have been called for short content")
	}
}

func TestWebFetchTool_LongContentSummarized(t *testing.T) {
	longContent := strings.Repeat("Lorem ipsum dolor sit amet. ", 200)
	mock := &mockWebSearcher{
		fetchResult: &websearch.FetchResult{
			URL:     "https://example.com/long",
			Content: longContent,
		},
	}
	mp := &mockProvider{
		chatResp: &provider.ChatResponse{
			Content: []provider.ContentBlock{
				{Type: provider.ContentText, Text: "This is a summary of the page."},
			},
			StopReason: provider.StopEndTurn,
		},
	}
	tool := NewWebFetchTool(WebToolDeps{WS: mock, Provider: mp, Model: "fast-model"})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com/long","question":"summarize"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "This is a summary of the page.") {
		t.Error("expected summary in output")
	}
	if strings.Contains(out, "Lorem ipsum") {
		t.Error("raw content should not appear in output")
	}
}

func TestWebFetchTool_SummarizerPrompt(t *testing.T) {
	longContent := strings.Repeat("x", 3000)
	mock := &mockWebSearcher{
		fetchResult: &websearch.FetchResult{
			URL:     "https://example.com",
			Content: longContent,
		},
	}
	mp := &mockProvider{
		chatResp: &provider.ChatResponse{
			Content: []provider.ContentBlock{
				{Type: provider.ContentText, Text: "summary"},
			},
			StopReason: provider.StopEndTurn,
		},
	}
	tool := NewWebFetchTool(WebToolDeps{WS: mock, Provider: mp, Model: "fast-model"})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","question":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sys := mp.lastReq.System
	if !strings.Contains(sys, "summarizer") {
		t.Error("system prompt should mention summarizer")
	}
	if !strings.Contains(sys, "125 characters") {
		t.Error("system prompt should mention 125 character quote limit")
	}
	if !strings.Contains(sys, "Do NOT follow") {
		t.Error("system prompt should instruct not to follow page instructions")
	}
}

func TestWebFetchTool_TruncatesLargeOutput(t *testing.T) {
	// Short content path but >1000 lines should still be truncated.
	longContent := strings.Repeat("line\n", 1500) // 1500 lines, <2000 chars threshold? No, 7500 chars > 2000.
	mock := &mockWebSearcher{
		fetchResult: &websearch.FetchResult{
			URL:     "https://example.com/big",
			Content: longContent,
		},
	}
	// Content > shortContentThreshold, so summarizer runs.
	mp := &mockProvider{
		chatResp: &provider.ChatResponse{
			Content: []provider.ContentBlock{
				{Type: provider.ContentText, Text: strings.Repeat("summary line\n", 1200)},
			},
			StopReason: provider.StopEndTurn,
		},
	}
	tool := NewWebFetchTool(WebToolDeps{WS: mock, Provider: mp, Model: "fast-model"})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com/big","question":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[truncated: showing 1000 of") {
		t.Error("expected truncation marker for >1000 lines of web fetch output")
	}
}

func TestWebFetchTool_EmptyURL(t *testing.T) {
	tool := NewWebFetchTool(WebToolDeps{})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"","question":"test"}`))
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestWebFetchTool_EmptyQuestion(t *testing.T) {
	tool := NewWebFetchTool(WebToolDeps{})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","question":""}`))
	if err == nil {
		t.Fatal("expected error for empty question")
	}
}

func TestWebFetchTool_SummarizerError(t *testing.T) {
	longContent := strings.Repeat("x", 3000)
	mock := &mockWebSearcher{
		fetchResult: &websearch.FetchResult{
			URL:     "https://example.com",
			Content: longContent,
		},
	}
	mp := &mockProvider{chatErr: errors.New("rate limited")}
	tool := NewWebFetchTool(WebToolDeps{WS: mock, Provider: mp, Model: "fast-model"})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","question":"test"}`))
	if err == nil {
		t.Fatal("expected error when summarizer fails")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected summarizer error, got: %v", err)
	}
}

func TestWebFetchTool_FetchError(t *testing.T) {
	mock := &mockWebSearcher{fetchErr: errors.New("connection refused")}
	tool := NewWebFetchTool(WebToolDeps{WS: mock})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"url":"https://example.com","question":"test"}`))
	if err == nil {
		t.Fatal("expected error from fetch")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected fetch error, got: %v", err)
	}
}
