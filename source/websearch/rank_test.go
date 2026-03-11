package websearch

import "testing"

func TestRankDeduplicates(t *testing.T) {
	results := []Result{
		{Title: "Page A", URL: "https://example.com/a", Source: "eng1"},
		{Title: "Page A", URL: "https://example.com/a", Source: "eng2"},
		{Title: "Page B", URL: "https://example.com/b", Source: "eng1"},
	}
	ranked := rankResults(results, "page")
	if len(ranked) != 2 {
		t.Fatalf("expected 2 results after dedup, got %d", len(ranked))
	}
}

func TestRankFrequencyBoost(t *testing.T) {
	results := []Result{
		{Title: "Unique", URL: "https://unique.com", Source: "eng1"},
		{Title: "Popular", URL: "https://popular.com", Source: "eng1"},
		{Title: "Popular", URL: "https://popular.com", Source: "eng2"},
	}
	ranked := rankResults(results, "test")
	if ranked[0].URL != "https://popular.com" {
		t.Errorf("expected popular (multi-backend) first, got %q", ranked[0].URL)
	}
}

func TestRankQueryRelevance(t *testing.T) {
	results := []Result{
		{Title: "Unrelated topic", URL: "https://a.com", Snippet: "nothing here", Source: "eng1"},
		{Title: "Go programming language", URL: "https://b.com", Snippet: "learn Go programming", Source: "eng1"},
	}
	ranked := rankResults(results, "Go programming")
	if ranked[0].URL != "https://b.com" {
		t.Errorf("expected query-relevant result first, got %q", ranked[0].URL)
	}
}

func TestRankTitleWeightHigherThanSnippet(t *testing.T) {
	results := []Result{
		{Title: "Other", URL: "https://a.com", Snippet: "contains golang info", Source: "eng1"},
		{Title: "Golang tutorial", URL: "https://b.com", Snippet: "a tutorial", Source: "eng1"},
	}
	ranked := rankResults(results, "golang")
	if ranked[0].URL != "https://b.com" {
		t.Errorf("expected title-match to rank higher, got %q", ranked[0].URL)
	}
}

func TestRankWikipediaPriority(t *testing.T) {
	results := []Result{
		{Title: "Regular result", URL: "https://regular.com", Source: "brave"},
		{Title: "Wikipedia article", URL: "https://en.wikipedia.org/wiki/Test", Source: "wikipedia"},
	}
	ranked := rankResults(results, "unrelated query")
	if ranked[0].Source != "wikipedia" {
		t.Errorf("expected wikipedia first, got %q", ranked[0].Source)
	}
}

func TestRankEmptyInput(t *testing.T) {
	ranked := rankResults(nil, "test")
	if len(ranked) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(ranked))
	}

	ranked = rankResults([]Result{}, "test")
	if len(ranked) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(ranked))
	}
}

func TestRankStableOrder(t *testing.T) {
	results := []Result{
		{Title: "First", URL: "https://a.com", Source: "eng1"},
		{Title: "Second", URL: "https://b.com", Source: "eng1"},
		{Title: "Third", URL: "https://c.com", Source: "eng1"},
	}
	ranked := rankResults(results, "unrelated")
	if ranked[0].Title != "First" || ranked[1].Title != "Second" || ranked[2].Title != "Third" {
		t.Error("expected stable order for equally-scored results")
	}
}
