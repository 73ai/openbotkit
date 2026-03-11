package websearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/priyanshujain/openbotkit/store"
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

func TestWebSearchStatusWithDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := store.Open(store.SQLiteConfig(dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Insert some search history rows.
	for _, q := range []string{"golang", "rust", "python"} {
		_, err := db.Exec("INSERT INTO search_history (query, result_count, backends, search_ms) VALUES (?, ?, ?, ?)",
			q, 5, "duckduckgo", 100)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	ws := New(Config{})
	st, err := ws.Status(context.Background(), db)
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
