package tools

import (
	"context"

	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/source/websearch"
)

// WebSearcher abstracts the websearch.WebSearch methods used by web tools.
type WebSearcher interface {
	Search(ctx context.Context, query string, opts websearch.SearchOptions) (*websearch.SearchResult, error)
	Fetch(ctx context.Context, url string, opts websearch.FetchOptions) (*websearch.FetchResult, error)
}

// WebToolDeps holds shared dependencies for web_search and web_fetch tools.
type WebToolDeps struct {
	WS       WebSearcher
	Provider provider.Provider
	Model    string
}
