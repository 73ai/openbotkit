package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/source/websearch"
)

// WebSearchTool searches the web and returns titles, URLs, and snippets.
type WebSearchTool struct {
	deps WebToolDeps
}

func NewWebSearchTool(deps WebToolDeps) *WebSearchTool {
	return &WebSearchTool{deps: deps}
}

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "Search the web and return titles, URLs, and snippets"
}
func (t *WebSearchTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Search query"
			},
			"max_results": {
				"type": "integer",
				"description": "Maximum number of results (default: 10)"
			}
		},
		"required": ["query"]
	}`)
}

type webSearchInput struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

func (t *WebSearchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in webSearchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}
	if in.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if in.MaxResults <= 0 {
		in.MaxResults = 10
	}

	result, err := t.deps.WS.Search(ctx, in.Query, websearch.SearchOptions{
		MaxResults: in.MaxResults,
	})
	if err != nil {
		return "", fmt.Errorf("web search: %w", err)
	}

	out := formatSearchResult(result)
	out = TruncateHead(out, MaxLinesWebSearch)
	out = TruncateBytes(out, MaxOutputBytes)
	return out, nil
}

func formatSearchResult(r *websearch.SearchResult) string {
	var b strings.Builder
	for i, res := range r.Results {
		fmt.Fprintf(&b, "[%d] %s\n    %s\n", i+1, res.Title, res.URL)
		if res.Snippet != "" {
			fmt.Fprintf(&b, "    %s\n", res.Snippet)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Found %d results via %s in %dms",
		r.Metadata.TotalResults,
		strings.Join(r.Metadata.Backends, ", "),
		r.Metadata.SearchTimeMs)
	return b.String()
}
