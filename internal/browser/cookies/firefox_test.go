package cookies

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestExtractFirefoxCookiesFromDB(t *testing.T) {
	dbPath := createFirefoxCookieDB(t, []firefoxCookieRow{
		{host: ".x.com", name: "auth_token", value: "ff-auth-123"},
		{host: ".x.com", name: "ct0", value: "ff-csrf-456"},
		{host: ".google.com", name: "NID", value: "google-nid"},
	})

	hosts := []string{".x.com", "x.com", ".twitter.com", "twitter.com"}
	names := []string{"auth_token", "ct0"}

	result, err := extractFirefoxCookiesFromDB(dbPath, hosts, names)
	if err != nil {
		t.Fatalf("extractFirefoxCookiesFromDB: %v", err)
	}

	if got := result["auth_token"]; got != "ff-auth-123" {
		t.Errorf("auth_token = %q, want %q", got, "ff-auth-123")
	}
	if got := result["ct0"]; got != "ff-csrf-456" {
		t.Errorf("ct0 = %q, want %q", got, "ff-csrf-456")
	}
	if _, ok := result["NID"]; ok {
		t.Error("NID should have been filtered out")
	}
}

func TestExtractFirefoxCookiesFromDB_Empty(t *testing.T) {
	dbPath := createFirefoxCookieDB(t, nil)

	hosts := []string{".x.com"}
	names := []string{"auth_token"}

	result, err := extractFirefoxCookiesFromDB(dbPath, hosts, names)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestExtractFirefoxCookiesFromDB_MissingDB(t *testing.T) {
	_, err := extractFirefoxCookiesFromDB("/nonexistent/path/cookies.sqlite", []string{".x.com"}, []string{"auth_token"})
	if err == nil {
		t.Fatal("expected error for missing DB")
	}
}

func TestExtractFirefoxCookiesFromDB_MultipleHosts(t *testing.T) {
	dbPath := createFirefoxCookieDB(t, []firefoxCookieRow{
		{host: ".twitter.com", name: "auth_token", value: "tw-auth"},
		{host: ".x.com", name: "ct0", value: "x-csrf"},
	})

	hosts := []string{".x.com", "x.com", ".twitter.com", "twitter.com"}
	names := []string{"auth_token", "ct0"}

	result, err := extractFirefoxCookiesFromDB(dbPath, hosts, names)
	if err != nil {
		t.Fatalf("extractFirefoxCookiesFromDB: %v", err)
	}

	if got := result["auth_token"]; got != "tw-auth" {
		t.Errorf("auth_token = %q, want %q", got, "tw-auth")
	}
	if got := result["ct0"]; got != "x-csrf" {
		t.Errorf("ct0 = %q, want %q", got, "x-csrf")
	}
}

// --- helpers ---

type firefoxCookieRow struct {
	host  string
	name  string
	value string
}

func createFirefoxCookieDB(t *testing.T, rows []firefoxCookieRow) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "cookies.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE moz_cookies (
		id INTEGER PRIMARY KEY,
		host TEXT NOT NULL,
		name TEXT NOT NULL,
		value TEXT NOT NULL,
		path TEXT DEFAULT '/',
		expiry INTEGER DEFAULT 0,
		isSecure INTEGER DEFAULT 0,
		isHttpOnly INTEGER DEFAULT 0,
		sameSite INTEGER DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	for _, r := range rows {
		_, err := db.Exec(`INSERT INTO moz_cookies (host, name, value, expiry) VALUES (?, ?, ?, ?)`,
			r.host, r.name, r.value, 1893456000) // 2030-01-01
		if err != nil {
			t.Fatalf("insert row: %v", err)
		}
	}

	return dbPath
}
