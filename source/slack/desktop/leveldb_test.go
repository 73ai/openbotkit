package desktop

import (
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestExtractTokenFromDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	db.Put([]byte("somekey"), []byte(`{"token":"xoxc-abc123-def456"}`), nil)
	db.Put([]byte("otherkey"), []byte("no token here"), nil)
	db.Close()

	token, err := extractTokenFromDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if token != "xoxc-abc123-def456" {
		t.Errorf("token = %q", token)
	}
}

func TestExtractTokenFromDB_NoToken(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	db.Put([]byte("key"), []byte("no token here"), nil)
	db.Close()

	token, err := extractTokenFromDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestExtractTokenFromDB_MissingDir(t *testing.T) {
	_, err := extractTokenFromDB("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestTokenRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`{"token":"xoxc-abc-123"}`, "xoxc-abc-123"},
		{`plain text with xoxc-longtoken-here inside`, "xoxc-longtoken-here"},
		{"no token", ""},
	}
	for _, tt := range tests {
		got := tokenRe.FindString(tt.input)
		if got != tt.want {
			t.Errorf("FindString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
