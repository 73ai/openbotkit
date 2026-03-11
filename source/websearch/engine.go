package websearch

import "context"

type Engine interface {
	Name() string
	Search(ctx context.Context, query string, opts SearchOptions) ([]Result, error)
	Priority() int
}
