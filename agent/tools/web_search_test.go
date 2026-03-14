package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/source/websearch"
)

type mockWebSearcher struct {
	searchResult *websearch.SearchResult
	searchErr    error
	fetchResult  *websearch.FetchResult
	fetchErr     error
	lastOpts     websearch.SearchOptions
}

func (m *mockWebSearcher) Search(_ context.Context, _ string, opts websearch.SearchOptions) (*websearch.SearchResult, error) {
	m.lastOpts = opts
	return m.searchResult, m.searchErr
}

func (m *mockWebSearcher) Fetch(_ context.Context, _ string, _ websearch.FetchOptions) (*websearch.FetchResult, error) {
	return m.fetchResult, m.fetchErr
}

func TestWebSearchTool_Name(t *testing.T) {
	tool := NewWebSearchTool(WebToolDeps{})
	if tool.Name() != "web_search" {
		t.Fatalf("expected name web_search, got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("expected non-empty description")
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema(), &schema); err != nil {
		t.Fatalf("invalid JSON schema: %v", err)
	}
}

func TestWebSearchTool_BasicSearch(t *testing.T) {
	mock := &mockWebSearcher{
		searchResult: &websearch.SearchResult{
			Query: "test query",
			Results: []websearch.Result{
				{Title: "Result One", URL: "https://one.example.com", Snippet: "First snippet"},
				{Title: "Result Two", URL: "https://two.example.com", Snippet: "Second snippet"},
			},
			Metadata: websearch.SearchMetadata{
				Backends:     []string{"duckduckgo"},
				SearchTimeMs: 150,
				TotalResults: 2,
			},
		},
	}
	tool := NewWebSearchTool(WebToolDeps{WS: mock})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test query"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "[1] Result One") {
		t.Error("expected numbered result 1")
	}
	if !strings.Contains(out, "https://one.example.com") {
		t.Error("expected first URL")
	}
	if !strings.Contains(out, "[2] Result Two") {
		t.Error("expected numbered result 2")
	}
	if !strings.Contains(out, "Found 2 results via duckduckgo in 150ms") {
		t.Error("expected metadata line")
	}
}

func TestWebSearchTool_EmptyQuery(t *testing.T) {
	tool := NewWebSearchTool(WebToolDeps{})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":""}`))
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestWebSearchTool_InvalidJSON(t *testing.T) {
	tool := NewWebSearchTool(WebToolDeps{})
	_, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestWebSearchTool_SearchError(t *testing.T) {
	mock := &mockWebSearcher{searchErr: errors.New("backend down")}
	tool := NewWebSearchTool(WebToolDeps{WS: mock})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test"}`))
	if err == nil {
		t.Fatal("expected error from backend")
	}
	if !strings.Contains(err.Error(), "backend down") {
		t.Fatalf("expected backend error, got: %v", err)
	}
}

func TestWebSearchTool_ZeroResults(t *testing.T) {
	mock := &mockWebSearcher{
		searchResult: &websearch.SearchResult{
			Query:   "obscure query",
			Results: nil,
			Metadata: websearch.SearchMetadata{
				Backends:     []string{"duckduckgo"},
				SearchTimeMs: 50,
				TotalResults: 0,
			},
		},
	}
	tool := NewWebSearchTool(WebToolDeps{WS: mock})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"obscure query"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Found 0 results") {
		t.Error("expected zero results metadata line")
	}
}

func TestWebSearchTool_CustomMaxResults(t *testing.T) {
	mock := &mockWebSearcher{
		searchResult: &websearch.SearchResult{
			Metadata: websearch.SearchMetadata{Backends: []string{"test"}},
		},
	}
	tool := NewWebSearchTool(WebToolDeps{WS: mock})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test","max_results":5}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastOpts.MaxResults != 5 {
		t.Fatalf("expected max_results=5, got %d", mock.lastOpts.MaxResults)
	}
}

func TestWebSearchTool_TruncatesLargeOutput(t *testing.T) {
	// Generate >500 results to exceed the line limit.
	results := make([]websearch.Result, 600)
	for i := range results {
		results[i] = websearch.Result{
			Title: "Result", URL: "https://example.com", Snippet: "Snippet text here",
		}
	}
	mock := &mockWebSearcher{
		searchResult: &websearch.SearchResult{
			Query:   "big",
			Results: results,
			Metadata: websearch.SearchMetadata{
				Backends: []string{"test"}, TotalResults: 600,
			},
		},
	}
	tool := NewWebSearchTool(WebToolDeps{WS: mock})

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"big"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[truncated: showing 500 of") {
		t.Error("expected truncation marker for >500 lines of search results")
	}
}

func TestWebSearchTool_DefaultMaxResults(t *testing.T) {
	mock := &mockWebSearcher{
		searchResult: &websearch.SearchResult{
			Metadata: websearch.SearchMetadata{Backends: []string{"test"}},
		},
	}
	tool := NewWebSearchTool(WebToolDeps{WS: mock})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.lastOpts.MaxResults != 10 {
		t.Fatalf("expected default max_results=10, got %d", mock.lastOpts.MaxResults)
	}
}
