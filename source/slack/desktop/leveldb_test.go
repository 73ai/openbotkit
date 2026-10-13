package desktop

import (
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestExtractTokensFromDB_MultipleWorkspaces(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	db.Put([]byte("teams"), []byte(`{"T1":{"token":"xoxc-team-one"},"T2":{"token":"xoxc-team-two"}}`), nil)
	db.Put([]byte("third"), []byte(`{"token":"xoxc-team-three"}`), nil)
	db.Close()

	tokens, err := extractTokensFromDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	for _, want := range []string{"xoxc-team-one", "xoxc-team-two", "xoxc-team-three"} {
		if !found[want] {
			t.Errorf("missing %q in %v", want, tokens)
		}
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

	tokens, err := extractTokensFromDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected no tokens, got %v", tokens)
	}
}

func TestExtractTokenFromDB_MissingDir(t *testing.T) {
	_, err := extractTokensFromDB("/nonexistent/path")
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
