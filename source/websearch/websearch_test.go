package websearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/store"
)

func TestWebSearchName(t *testing.T) {
	ws := New(Config{})
	if ws.Name() != "websearch" {
		t.Fatalf("expected 'websearch', got %q", ws.Name())
	}
}

func TestWebSearchStatusNoDB(t *testing.T) {
	ws := New(Config{})
	st, err := ws.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Connected {
		t.Error("expected Connected=true")
	}
	if st.ItemCount != 0 {
		t.Errorf("expected ItemCount=0, got %d", st.ItemCount)
	}
}

func TestWithDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	ws := New(Config{}, WithDB(db))
	if ws.db == nil {
		t.Fatal("expected db to be set")
	}
}

func TestNewWithoutOptions(t *testing.T) {
	ws := New(Config{})
	if ws.db != nil {
		t.Fatal("expected db to be nil")
	}
}

func TestWebSearchStatusWithHistory(t *testing.T) {
	histPath := filepath.Join(t.TempDir(), "search_history.jsonl")

	for _, q := range []string{"golang", "rust", "python"} {
		putSearchHistory(histPath, q, "web", 5, []string{"duckduckgo"}, 100)
	}

	ws := New(Config{}, WithHistoryPath(histPath))
	st, err := ws.Status(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !st.Connected {
		t.Error("expected Connected=true")
	}
	if st.ItemCount != 3 {
		t.Errorf("expected ItemCount=3, got %d", st.ItemCount)
	}
}

func TestCacheTTLDefault(t *testing.T) {
	ws := New(Config{})
	if ws.cacheTTL() != 15*time.Minute {
		t.Errorf("expected 15m default, got %v", ws.cacheTTL())
	}
}

func TestCacheTTLFromConfig(t *testing.T) {
	ws := New(Config{WebSearch: &config.WebSearchConfig{CacheTTL: "30m"}})
	if ws.cacheTTL() != 30*time.Minute {
		t.Errorf("expected 30m, got %v", ws.cacheTTL())
	}
}

func TestCacheTTLInvalidFallsBack(t *testing.T) {
	ws := New(Config{WebSearch: &config.WebSearchConfig{CacheTTL: "invalid"}})
	if ws.cacheTTL() != 15*time.Minute {
		t.Errorf("expected 15m fallback, got %v", ws.cacheTTL())
	}
}
