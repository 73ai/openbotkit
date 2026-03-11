package websearch

import "context"

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
