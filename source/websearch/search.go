package websearch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	defaultMaxResults = 10
	defaultRegion     = "us-en"
)

func (w *WebSearch) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("empty search query")
	}

	if opts.MaxResults <= 0 {
		opts.MaxResults = defaultMaxResults
	}
	if opts.Region == "" {
		opts.Region = defaultRegion
	}

	if !opts.NoCache {
		key := cacheKey(query, "web", opts.Backend, opts.Region, opts.TimeLimit)
		if cached, ok := getSearchCache(w.db, key, w.cacheTTL()); ok {
			return cached, nil
		}
	}

	client := w.httpClient()
	engines := buildEngines(client, opts.Backend, w.configuredBackends())
	if len(engines) == 0 {
		return nil, fmt.Errorf("unknown backend: %q", opts.Backend)
	}

	result, err := w.searchWithEngines(ctx, query, opts, engines)
	if err != nil {
		return nil, err
	}

	if !opts.NoCache {
		key := cacheKey(query, "web", opts.Backend, opts.Region, opts.TimeLimit)
		putSearchCache(w.db, key, query, "web", result.Results)
	}

	putSearchHistory(w.db, query, "web", result.Metadata.TotalResults, result.Metadata.Backends, result.Metadata.SearchTimeMs)

	return result, nil
}

func (w *WebSearch) searchWithEngines(ctx context.Context, query string, opts SearchOptions, engines []Engine) (*SearchResult, error) {
	if opts.MaxResults <= 0 {
		opts.MaxResults = defaultMaxResults
	}

	// Sort by priority descending (higher priority first).
	sort.Slice(engines, func(i, j int) bool {
		return engines[i].Priority() > engines[j].Priority()
	})

	start := time.Now()
	var allResults []Result
	var backends []string
	var lastErr error

	for _, eng := range engines {
		if !w.health.IsHealthy(eng.Name()) {
			slog.Info("skipping unhealthy backend", "engine", eng.Name())
			continue
		}
		results, err := eng.Search(ctx, query, opts)
		if err != nil {
			lastErr = err
			w.health.RecordFailure(eng.Name())
			slog.Warn("search engine failed", "engine", eng.Name(), "error", err)
			continue
		}
		w.health.RecordSuccess(eng.Name())
		allResults = append(allResults, results...)
		backends = append(backends, eng.Name())
	}

	if len(allResults) == 0 && lastErr != nil {
		return nil, fmt.Errorf("all backends failed: %w", lastErr)
	}

	allResults = rankResults(allResults, query)

	if len(allResults) > opts.MaxResults {
		allResults = allResults[:opts.MaxResults]
	}

	elapsed := time.Since(start).Milliseconds()

	return &SearchResult{
		Query:   query,
		Results: allResults,
		Metadata: SearchMetadata{
			Backends:     backends,
			SearchTimeMs: elapsed,
			TotalResults: len(allResults),
		},
	}, nil
}

func buildEngines(client HTTPDoer, backend string, configured []string) []Engine {
	switch backend {
	case "", "auto":
		all := []Engine{
			NewDuckDuckGo(client),
			NewBrave(client),
			NewMojeek(client),
			NewWikipedia(client),
		}
		return filterEngines(all, configured)
	case "duckduckgo":
		return []Engine{NewDuckDuckGo(client)}
	case "brave":
		return []Engine{NewBrave(client)}
	case "mojeek":
		return []Engine{NewMojeek(client)}
	case "yahoo":
		return []Engine{NewYahoo(client)}
	case "yandex":
		return []Engine{NewYandex(client)}
	case "google":
		return []Engine{NewGoogle(client)}
	case "wikipedia":
		return []Engine{NewWikipedia(client)}
	case "bing":
		return []Engine{NewBing(client)}
	default:
		return nil
	}
}

func filterEngines(engines []Engine, allowed []string) []Engine {
	if len(allowed) == 0 {
		return engines
	}
	set := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		set[name] = true
	}
	var out []Engine
	for _, eng := range engines {
		if set[eng.Name()] {
			out = append(out, eng)
		}
	}
	return out
}

func (w *WebSearch) News(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("empty search query")
	}

	if opts.MaxResults <= 0 {
		opts.MaxResults = defaultMaxResults
	}
	if opts.Region == "" {
		opts.Region = defaultRegion
	}

	if !opts.NoCache {
		key := cacheKey(query, "news", opts.Backend, opts.Region, opts.TimeLimit)
		if cached, ok := getSearchCache(w.db, key, w.cacheTTL()); ok {
			return cached, nil
		}
	}

	client := w.httpClient()
	engines := buildNewsEngines(client, opts.Backend, w.configuredBackends())
	if len(engines) == 0 {
		return nil, fmt.Errorf("unknown or non-news backend: %q", opts.Backend)
	}

	result, err := w.newsWithEngines(ctx, query, opts, engines)
	if err != nil {
		return nil, err
	}

	if !opts.NoCache {
		key := cacheKey(query, "news", opts.Backend, opts.Region, opts.TimeLimit)
		putSearchCache(w.db, key, query, "news", result.Results)
	}

	putSearchHistory(w.db, query, "news", result.Metadata.TotalResults, result.Metadata.Backends, result.Metadata.SearchTimeMs)

	return result, nil
}

func (w *WebSearch) newsWithEngines(ctx context.Context, query string, opts SearchOptions, engines []NewsEngine) (*SearchResult, error) {
	if opts.MaxResults <= 0 {
		opts.MaxResults = defaultMaxResults
	}

	sort.Slice(engines, func(i, j int) bool {
		return engines[i].Priority() > engines[j].Priority()
	})

	start := time.Now()
	var allResults []Result
	var backends []string
	var lastErr error

	for _, eng := range engines {
		if !w.health.IsHealthy(eng.Name()) {
			slog.Info("skipping unhealthy news backend", "engine", eng.Name())
			continue
		}
		results, err := eng.News(ctx, query, opts)
		if err != nil {
			lastErr = err
			w.health.RecordFailure(eng.Name())
			slog.Warn("news engine failed", "engine", eng.Name(), "error", err)
			continue
		}
		w.health.RecordSuccess(eng.Name())
		allResults = append(allResults, results...)
		backends = append(backends, eng.Name())
	}

	if len(allResults) == 0 && lastErr != nil {
		return nil, fmt.Errorf("all news backends failed: %w", lastErr)
	}

	allResults = rankResults(allResults, query)

	if len(allResults) > opts.MaxResults {
		allResults = allResults[:opts.MaxResults]
	}

	elapsed := time.Since(start).Milliseconds()

	return &SearchResult{
		Query:   query,
		Results: allResults,
		Metadata: SearchMetadata{
			Backends:     backends,
			SearchTimeMs: elapsed,
			TotalResults: len(allResults),
		},
	}, nil
}

func buildNewsEngines(client HTTPDoer, backend string, configured []string) []NewsEngine {
	switch backend {
	case "", "auto":
		all := []NewsEngine{
			NewDuckDuckGo(client),
			NewYahoo(client),
		}
		return filterNewsEngines(all, configured)
	case "duckduckgo":
		return []NewsEngine{NewDuckDuckGo(client)}
	case "yahoo":
		return []NewsEngine{NewYahoo(client)}
	default:
		return nil
	}
}

func filterNewsEngines(engines []NewsEngine, allowed []string) []NewsEngine {
	if len(allowed) == 0 {
		return engines
	}
	set := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		set[name] = true
	}
	var out []NewsEngine
	for _, eng := range engines {
		if set[eng.Name()] {
			out = append(out, eng)
		}
	}
	return out
}

func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	return strings.TrimSuffix(u.String(), "/")
}
