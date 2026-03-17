package tools

import (
	"context"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/source/websearch"
	"github.com/73ai/openbotkit/store"
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

// WebSearchSetup holds the inputs needed to create a websearch instance.
type WebSearchSetup struct {
	WSConfig *config.WebSearchConfig
	DSN      string // pre-resolved data DSN
}

// NewWebSearchInstance creates a websearch.WebSearch with an optional cache DB.
// Returns the WebSearch instance and the DB handle (nil if no cache).
// The caller is responsible for closing the DB handle.
func NewWebSearchInstance(s WebSearchSetup) (*websearch.WebSearch, *store.DB) {
	if s.WSConfig == nil {
		return websearch.New(websearch.Config{}), nil
	}
	var (
		opts []websearch.Option
		db   *store.DB
	)
	if s.DSN != "" {
		if err := config.EnsureSourceDir("websearch"); err == nil {
			opened, err := store.Open(store.Config{
				Driver: s.WSConfig.Storage.Driver,
				DSN:    s.DSN,
			})
			if err == nil {
				db = opened
				opts = append(opts, websearch.WithDB(db))
			}
		}
	}
	return websearch.New(websearch.Config{WebSearch: s.WSConfig}, opts...), db
}

// ResolveFastProvider returns the fast-tier provider and model, falling back
// to the given defaults if the fast tier is not configured.
func ResolveFastProvider(models *config.ModelsConfig, reg *provider.Registry, defaultP provider.Provider, defaultModel string) (provider.Provider, string) {
	if models != nil && models.Fast != "" {
		provName, model, err := provider.ParseModelSpec(models.Fast)
		if err == nil {
			if p, ok := reg.Get(provName); ok {
				return p, model
			}
		}
	}
	return defaultP, defaultModel
}

// ResolveNanoProvider returns the nano-tier provider and model.
// Cascades: nano → fast → default.
func ResolveNanoProvider(models *config.ModelsConfig, reg *provider.Registry, defaultP provider.Provider, defaultModel string) (provider.Provider, string) {
	if models != nil && models.Nano != "" {
		provName, model, err := provider.ParseModelSpec(models.Nano)
		if err == nil {
			if p, ok := reg.Get(provName); ok {
				return p, model
			}
		}
	}
	return ResolveFastProvider(models, reg, defaultP, defaultModel)
}
