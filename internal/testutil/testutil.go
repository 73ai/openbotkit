package testutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/internal/envload"
	"github.com/priyanshujain/openbotkit/memory"
	"github.com/priyanshujain/openbotkit/store"
)

// LoadEnv reads a .env file and sets environment variables for the test.
// It walks up from the working directory looking for .env.
func LoadEnv(t *testing.T) {
	t.Helper()
	envload.Load(t)
}

// TestDB creates an in-memory SQLite DB with cleanup.
func TestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestConfig returns a Config pointing at a temp dir with mode set.
func TestConfig(t *testing.T, mode config.Mode) *config.Config {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	// Ensure all source dirs exist
	for _, src := range []string{"gmail", "whatsapp", "history", "user_memory", "applenotes"} {
		if err := os.MkdirAll(filepath.Join(dir, src), 0700); err != nil {
			t.Fatalf("mkdir %s: %v", src, err)
		}
	}

	cfg := config.Default()
	cfg.Mode = mode
	return cfg
}

// TestServerResult holds the test server and its config.
type TestServerResult struct {
	Server *httptest.Server
	Config *config.Config
	URL    string
}

// SeedMemoryDB creates and migrates the memory database for a config.
func SeedMemoryDB(t *testing.T, cfg *config.Config) {
	t.Helper()
	dsn := cfg.UserMemoryDataDSN()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	if err := memory.Migrate(db); err != nil {
		t.Fatalf("migrate memory: %v", err)
	}
	db.Close()
}

// RequireGeminiKey loads .env and returns the Gemini API key, skipping if unavailable.
func RequireGeminiKey(t *testing.T) string {
	t.Helper()
	LoadEnv(t)
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not set — skipping")
	}
	return key
}

// AuthenticatedRequest creates an HTTP request with basic auth.
func AuthenticatedRequest(method, url string, body *strings.Reader) *http.Request {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, url, body)
	} else {
		req, _ = http.NewRequest(method, url, nil)
	}
	req.SetBasicAuth("test", "test")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}
