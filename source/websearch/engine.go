package websearch

import (
	"context"
	"net/http"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Engine interface {
	Name() string
	Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error)
	Priority() int
}

type NewsEngine interface {
	Name() string
	News(ctx context.Context, query string, opts SearchOptions) ([]Result, error)
	Priority() int
}
