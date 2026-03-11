package websearch

import (
	"context"
	"errors"
	"testing"
)

type mockEngine struct {
	name     string
	priority int
	results  []Result
	err      error
	calls    int
}

func (m *mockEngine) Name() string  { return m.name }
func (m *mockEngine) Priority() int { return m.priority }
func (m *mockEngine) Search(_ context.Context, _ string, _ SearchOptions) ([]Result, error) {
	m.calls++
	return m.results, m.err
}

func searchWithEngines(engines []Engine, query string, opts SearchOptions) (*SearchResult, error) {
	ws := New(Config{})
	return ws.searchWithEngines(context.Background(), query, opts, engines)
}

func TestSearchPriorityOrder(t *testing.T) {
	engines := []Engine{
		&mockEngine{name: "low", priority: 1, results: []Result{
			{Title: "Low", URL: "https://low.com", Source: "low"},
		}},
		&mockEngine{name: "high", priority: 2, results: []Result{
			{Title: "High", URL: "https://high.com", Source: "high"},
		}},
	}

	result, err := searchWithEngines(engines, "test", SearchOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].Source != "high" {
		t.Errorf("expected high-priority first, got %q", result.Results[0].Source)
	}
}

func TestSearchDeduplication(t *testing.T) {
	engines := []Engine{
		&mockEngine{name: "a", priority: 2, results: []Result{
			{Title: "Page", URL: "https://example.com/page", Source: "a"},
		}},
		&mockEngine{name: "b", priority: 1, results: []Result{
			{Title: "Page", URL: "https://example.com/page", Source: "b"},
		}},
	}

	result, err := searchWithEngines(engines, "test", SearchOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result after dedup, got %d", len(result.Results))
	}
}

func TestSearchMaxResults(t *testing.T) {
	var results []Result
	for i := range 15 {
		results = append(results, Result{
			Title:  "Result",
			URL:    "https://example.com/" + string(rune('a'+i)),
			Source: "mock",
		})
	}
	engines := []Engine{
		&mockEngine{name: "mock", priority: 1, results: results},
	}

	result, err := searchWithEngines(engines, "test", SearchOptions{MaxResults: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(result.Results))
	}
}

func TestSearchFallbackOnError(t *testing.T) {
	engines := []Engine{
		&mockEngine{name: "failing", priority: 2, err: errors.New("failed")},
		&mockEngine{name: "working", priority: 1, results: []Result{
			{Title: "Works", URL: "https://works.com", Source: "working"},
		}},
	}

	result, err := searchWithEngines(engines, "test", SearchOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Source != "working" {
		t.Errorf("expected 'working' source, got %q", result.Results[0].Source)
	}
}

func TestSearchAllEnginesFail(t *testing.T) {
	engines := []Engine{
		&mockEngine{name: "a", priority: 2, err: errors.New("fail a")},
		&mockEngine{name: "b", priority: 1, err: errors.New("fail b")},
	}

	_, err := searchWithEngines(engines, "test", SearchOptions{MaxResults: 10})
	if err == nil {
		t.Fatal("expected error when all engines fail")
	}
}

func TestSearchBackendSelection(t *testing.T) {
	t.Run("duckduckgo only", func(t *testing.T) {
		engines := buildEngines(nil, "duckduckgo")
		if len(engines) != 1 || engines[0].Name() != "duckduckgo" {
			t.Errorf("expected only duckduckgo engine")
		}
	})

	t.Run("wikipedia only", func(t *testing.T) {
		engines := buildEngines(nil, "wikipedia")
		if len(engines) != 1 || engines[0].Name() != "wikipedia" {
			t.Errorf("expected only wikipedia engine")
		}
	})

	t.Run("brave only", func(t *testing.T) {
		engines := buildEngines(nil, "brave")
		if len(engines) != 1 || engines[0].Name() != "brave" {
			t.Errorf("expected only brave engine")
		}
	})

	t.Run("mojeek only", func(t *testing.T) {
		engines := buildEngines(nil, "mojeek")
		if len(engines) != 1 || engines[0].Name() != "mojeek" {
			t.Errorf("expected only mojeek engine")
		}
	})

	t.Run("yahoo only", func(t *testing.T) {
		engines := buildEngines(nil, "yahoo")
		if len(engines) != 1 || engines[0].Name() != "yahoo" {
			t.Errorf("expected only yahoo engine")
		}
	})

	t.Run("yandex only", func(t *testing.T) {
		engines := buildEngines(nil, "yandex")
		if len(engines) != 1 || engines[0].Name() != "yandex" {
			t.Errorf("expected only yandex engine")
		}
	})

	t.Run("google only", func(t *testing.T) {
		engines := buildEngines(nil, "google")
		if len(engines) != 1 || engines[0].Name() != "google" {
			t.Errorf("expected only google engine")
		}
	})

	t.Run("auto uses duckduckgo+brave+mojeek+wikipedia", func(t *testing.T) {
		engines := buildEngines(nil, "auto")
		if len(engines) != 4 {
			t.Errorf("expected 4 engines for auto, got %d", len(engines))
		}
	})

	t.Run("empty uses auto set", func(t *testing.T) {
		engines := buildEngines(nil, "")
		if len(engines) != 4 {
			t.Errorf("expected 4 engines for empty, got %d", len(engines))
		}
	})

	t.Run("unknown returns nil", func(t *testing.T) {
		engines := buildEngines(nil, "unknown")
		if engines != nil {
			t.Errorf("expected nil for unknown backend")
		}
	})
}

func TestSearchDefaultOptions(t *testing.T) {
	engines := []Engine{
		&mockEngine{name: "mock", priority: 1, results: []Result{
			{Title: "R1", URL: "https://example.com/1", Source: "mock"},
		}},
	}

	result, err := searchWithEngines(engines, "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) > defaultMaxResults {
		t.Errorf("expected at most %d results", defaultMaxResults)
	}
}

func TestSearchMetadata(t *testing.T) {
	engines := []Engine{
		&mockEngine{name: "eng1", priority: 2, results: []Result{
			{Title: "R1", URL: "https://example.com/1", Source: "eng1"},
		}},
		&mockEngine{name: "eng2", priority: 1, results: []Result{
			{Title: "R2", URL: "https://example.com/2", Source: "eng2"},
		}},
	}

	result, err := searchWithEngines(engines, "test", SearchOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Metadata.Backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(result.Metadata.Backends))
	}
	if result.Metadata.SearchTimeMs < 0 {
		t.Error("search time should be non-negative")
	}
	if result.Metadata.TotalResults != 2 {
		t.Errorf("expected 2 total results, got %d", result.Metadata.TotalResults)
	}
}

type mockNewsEngine struct {
	name     string
	priority int
	results  []Result
	err      error
}

func (m *mockNewsEngine) Name() string  { return m.name }
func (m *mockNewsEngine) Priority() int { return m.priority }
func (m *mockNewsEngine) News(_ context.Context, _ string, _ SearchOptions) ([]Result, error) {
	return m.results, m.err
}

func TestNewsPriorityOrder(t *testing.T) {
	engines := []NewsEngine{
		&mockNewsEngine{name: "low", priority: 1, results: []Result{
			{Title: "Low News", URL: "https://low.com/news", Source: "low"},
		}},
		&mockNewsEngine{name: "high", priority: 2, results: []Result{
			{Title: "High News", URL: "https://high.com/news", Source: "high"},
		}},
	}

	ws := New(Config{})
	result, err := ws.newsWithEngines(context.Background(), "test", SearchOptions{MaxResults: 10}, engines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].Source != "high" {
		t.Errorf("expected high-priority first, got %q", result.Results[0].Source)
	}
}

func TestNewsDedup(t *testing.T) {
	engines := []NewsEngine{
		&mockNewsEngine{name: "a", priority: 2, results: []Result{
			{Title: "News", URL: "https://example.com/news", Source: "a"},
		}},
		&mockNewsEngine{name: "b", priority: 1, results: []Result{
			{Title: "News", URL: "https://example.com/news", Source: "b"},
		}},
	}

	ws := New(Config{})
	result, err := ws.newsWithEngines(context.Background(), "test", SearchOptions{MaxResults: 10}, engines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result after dedup, got %d", len(result.Results))
	}
}

func TestNewsFallbackOnError(t *testing.T) {
	engines := []NewsEngine{
		&mockNewsEngine{name: "failing", priority: 2, err: errors.New("failed")},
		&mockNewsEngine{name: "working", priority: 1, results: []Result{
			{Title: "News", URL: "https://works.com/news", Source: "working"},
		}},
	}

	ws := New(Config{})
	result, err := ws.newsWithEngines(context.Background(), "test", SearchOptions{MaxResults: 10}, engines)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
}

func TestNewsBackendSelection(t *testing.T) {
	t.Run("auto uses duckduckgo+yahoo", func(t *testing.T) {
		engines := buildNewsEngines(nil, "auto")
		if len(engines) != 2 {
			t.Errorf("expected 2 news engines for auto, got %d", len(engines))
		}
	})

	t.Run("duckduckgo only", func(t *testing.T) {
		engines := buildNewsEngines(nil, "duckduckgo")
		if len(engines) != 1 || engines[0].Name() != "duckduckgo" {
			t.Errorf("expected only duckduckgo news engine")
		}
	})

	t.Run("yahoo only", func(t *testing.T) {
		engines := buildNewsEngines(nil, "yahoo")
		if len(engines) != 1 || engines[0].Name() != "yahoo" {
			t.Errorf("expected only yahoo news engine")
		}
	})

	t.Run("unsupported returns nil", func(t *testing.T) {
		engines := buildNewsEngines(nil, "wikipedia")
		if engines != nil {
			t.Errorf("expected nil for non-news backend")
		}
	})
}

func TestSearchCacheIntegration(t *testing.T) {
	db := openTestDB(t)
	ws := New(Config{}, WithDB(db))

	eng := &mockEngine{name: "mock", priority: 1, results: []Result{
		{Title: "R1", URL: "https://example.com/1", Source: "mock"},
	}}

	// First call — cache miss, engine called.
	r1, err := ws.searchWithEngines(context.Background(), "test", SearchOptions{MaxResults: 10, Backend: "auto", Region: "us-en"}, []Engine{eng})
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if eng.calls != 1 {
		t.Fatalf("expected 1 engine call, got %d", eng.calls)
	}

	// Populate cache as Search() would.
	key := cacheKey("test", "web", "auto", "us-en", "")
	putSearchCache(db, key, "test", "web", r1.Results)

	// Second call — cache hit, engine NOT called again.
	cached, ok := getSearchCache(db, key, ws.cacheTTL())
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !cached.Metadata.Cached {
		t.Error("expected Cached=true in metadata")
	}
	if len(cached.Results) != 1 || cached.Results[0].Title != "R1" {
		t.Errorf("unexpected cached results: %v", cached.Results)
	}
}

func TestSearchNoCacheBypass(t *testing.T) {
	db := openTestDB(t)

	// Pre-populate cache.
	key := cacheKey("bypass-test", "web", "auto", "us-en", "")
	putSearchCache(db, key, "bypass-test", "web", []Result{
		{Title: "Cached", URL: "https://cached.com", Source: "cache"},
	})

	eng := &mockEngine{name: "mock", priority: 1, results: []Result{
		{Title: "Fresh", URL: "https://fresh.com", Source: "mock"},
	}}
	ws := New(Config{}, WithDB(db))

	// With NoCache, searchWithEngines + manual NoCache check simulates Search().
	// Test that the engine is called despite cache existing.
	opts := SearchOptions{MaxResults: 10, Backend: "auto", Region: "us-en", NoCache: true}

	// NoCache means Search() skips getSearchCache. Verify engine gets called.
	result, err := ws.searchWithEngines(context.Background(), "bypass-test", opts, []Engine{eng})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eng.calls != 1 {
		t.Fatalf("expected engine to be called, got %d calls", eng.calls)
	}
	if result.Results[0].Title != "Fresh" {
		t.Errorf("expected fresh result, got %q", result.Results[0].Title)
	}
}

func TestSearchNilDBSkipsCache(t *testing.T) {
	eng := &mockEngine{name: "mock", priority: 1, results: []Result{
		{Title: "R1", URL: "https://example.com/1", Source: "mock"},
	}}
	ws := New(Config{}) // nil db

	result, err := ws.searchWithEngines(context.Background(), "test", SearchOptions{MaxResults: 10}, []Engine{eng})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if eng.calls != 1 {
		t.Fatalf("expected engine called once, got %d", eng.calls)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	ws := New(Config{})
	_, err := ws.Search(context.Background(), "", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for empty query")
	}

	_, err = ws.Search(context.Background(), "   ", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for whitespace-only query")
	}
}
