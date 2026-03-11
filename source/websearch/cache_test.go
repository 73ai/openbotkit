package websearch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cache_test.db")

	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db
}

func TestCacheKeyDeterminism(t *testing.T) {
	k1 := cacheKey("golang", "web", "auto", "us-en", "", 0)
	k2 := cacheKey("golang", "web", "auto", "us-en", "", 0)
	if k1 != k2 {
		t.Errorf("same inputs should produce same key: %q != %q", k1, k2)
	}

	k3 := cacheKey("golang", "news", "auto", "us-en", "", 0)
	if k1 == k3 {
		t.Errorf("different category should produce different key")
	}

	k4 := cacheKey("rust", "web", "auto", "us-en", "", 0)
	if k1 == k4 {
		t.Errorf("different query should produce different key")
	}

	k5 := cacheKey("golang", "web", "auto", "us-en", "", 2)
	if k1 == k5 {
		t.Errorf("different page should produce different key")
	}
}

func TestSearchCacheRoundTrip(t *testing.T) {
	db := openTestDB(t)

	key := cacheKey("golang", "web", "auto", "us-en", "", 0)
	results := []Result{
		{Title: "Go", URL: "https://go.dev", Snippet: "Go lang", Source: "duckduckgo"},
	}

	putSearchCache(db, key, "golang", "web", results)

	got, ok := getSearchCache(db, key, 15*time.Minute)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Query != "golang" {
		t.Errorf("expected query 'golang', got %q", got.Query)
	}
	if len(got.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got.Results))
	}
	if got.Results[0].Title != "Go" {
		t.Errorf("expected title 'Go', got %q", got.Results[0].Title)
	}
	if !got.Metadata.Cached {
		t.Error("expected Cached=true")
	}
}

func TestSearchCacheTTLExpiry(t *testing.T) {
	db := openTestDB(t)

	key := cacheKey("test", "web", "auto", "us-en", "", 0)
	putSearchCache(db, key, "test", "web", []Result{{Title: "T", URL: "https://t.com", Source: "mock"}})

	// Manually backdate the entry.
	db.Exec("UPDATE search_cache SET created_at = datetime('now', '-1 hour') WHERE cache_key = ?", key)

	_, ok := getSearchCache(db, key, 30*time.Minute)
	if ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestSearchCacheMiss(t *testing.T) {
	db := openTestDB(t)

	_, ok := getSearchCache(db, "nonexistent", 15*time.Minute)
	if ok {
		t.Error("expected cache miss on empty DB")
	}
}

func TestSearchCacheNilDB(t *testing.T) {
	putSearchCache(nil, "key", "query", "web", []Result{{Title: "T"}})
	_, ok := getSearchCache(nil, "key", 15*time.Minute)
	if ok {
		t.Error("nil DB should return cache miss")
	}
}

func TestFetchCacheRoundTrip(t *testing.T) {
	db := openTestDB(t)

	result := &FetchResult{
		URL:        "https://example.com",
		Title:      "Example",
		Content:    "Hello world",
		StatusCode: 200,
	}

	putFetchCache(db, result, "markdown")

	got, ok := getFetchCache(db, "https://example.com", "markdown", 15*time.Minute)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Title != "Example" {
		t.Errorf("expected title 'Example', got %q", got.Title)
	}
	if got.Content != "Hello world" {
		t.Errorf("expected content 'Hello world', got %q", got.Content)
	}
	if !got.Cached {
		t.Error("expected Cached=true")
	}
}

func TestFetchCacheFormatMismatch(t *testing.T) {
	db := openTestDB(t)

	putFetchCache(db, &FetchResult{URL: "https://example.com", Title: "T", Content: "# Hello", StatusCode: 200}, "markdown")

	_, ok := getFetchCache(db, "https://example.com", "text", 15*time.Minute)
	if ok {
		t.Error("expected cache miss for different format")
	}
}

func TestFetchCacheTTLExpiry(t *testing.T) {
	db := openTestDB(t)

	putFetchCache(db, &FetchResult{URL: "https://test.com", Title: "T", Content: "C", StatusCode: 200}, "markdown")

	db.Exec("UPDATE fetch_cache SET fetched_at = datetime('now', '-1 hour') WHERE url = ?", "https://test.com")

	_, ok := getFetchCache(db, "https://test.com", "markdown", 30*time.Minute)
	if ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestFetchCacheNilDB(t *testing.T) {
	putFetchCache(nil, &FetchResult{URL: "https://test.com"}, "markdown")
	_, ok := getFetchCache(nil, "https://test.com", "markdown", 15*time.Minute)
	if ok {
		t.Error("nil DB should return cache miss")
	}
}

func TestPutSearchHistory(t *testing.T) {
	db := openTestDB(t)
	putSearchHistory(db, "golang", "web", 5, []string{"duckduckgo", "brave"}, 342)

	var query, category, backends string
	var count int
	var ms int64
	err := db.QueryRow("SELECT query, category, result_count, backends, search_ms FROM search_history").Scan(&query, &category, &count, &backends, &ms)
	if err != nil {
		t.Fatalf("query history: %v", err)
	}
	if query != "golang" || category != "web" || count != 5 || backends != "duckduckgo,brave" || ms != 342 {
		t.Errorf("unexpected row: query=%q category=%q count=%d backends=%q ms=%d", query, category, count, backends, ms)
	}
}

func TestClearCachesExported(t *testing.T) {
	db := openTestDB(t)
	ws := New(Config{}, WithDB(db))

	putSearchCache(db, "key1", "q", "web", []Result{{Title: "T", URL: "https://t.com"}})

	if err := ws.ClearCaches(); err != nil {
		t.Fatalf("ClearCaches: %v", err)
	}

	_, ok := getSearchCache(db, "key1", 15*time.Minute)
	if ok {
		t.Error("search cache should be empty after ClearCaches")
	}
}

func TestClearAllCaches(t *testing.T) {
	db := openTestDB(t)

	putSearchCache(db, "key1", "q", "web", []Result{{Title: "T", URL: "https://t.com"}})
	putFetchCache(db, &FetchResult{URL: "https://f.com", Title: "F", Content: "C", StatusCode: 200}, "markdown")

	if err := clearAllCaches(db); err != nil {
		t.Fatalf("clear caches: %v", err)
	}

	_, ok := getSearchCache(db, "key1", 15*time.Minute)
	if ok {
		t.Error("search cache should be empty after clear")
	}
	_, ok = getFetchCache(db, "https://f.com", "markdown", 15*time.Minute)
	if ok {
		t.Error("fetch cache should be empty after clear")
	}
}
