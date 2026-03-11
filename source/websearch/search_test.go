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
}

func (m *mockEngine) Name() string  { return m.name }
func (m *mockEngine) Priority() int { return m.priority }
func (m *mockEngine) Search(_ context.Context, _ string, _ SearchOptions) ([]Result, error) {
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

	t.Run("auto uses both", func(t *testing.T) {
		engines := buildEngines(nil, "auto")
		if len(engines) != 2 {
			t.Errorf("expected 2 engines for auto, got %d", len(engines))
		}
	})

	t.Run("empty uses both", func(t *testing.T) {
		engines := buildEngines(nil, "")
		if len(engines) != 2 {
			t.Errorf("expected 2 engines for empty, got %d", len(engines))
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
