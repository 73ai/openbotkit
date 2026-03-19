package learnings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveNew(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	if err := st.Save("Go Basics", []string{"goroutines are lightweight threads"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "go-basics.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "# Go Basics") {
		t.Error("expected heading")
	}
	if !strings.Contains(string(data), "- goroutines are lightweight threads") {
		t.Error("expected bullet")
	}
}

func TestSaveAppend(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	st.Save("Go", []string{"first bullet"})
	st.Save("Go", []string{"second bullet"})

	data, _ := os.ReadFile(filepath.Join(dir, "go.md"))
	content := string(data)
	if strings.Count(content, "# Go") != 1 {
		t.Error("expected exactly one heading")
	}
	if !strings.Contains(content, "- first bullet") || !strings.Contains(content, "- second bullet") {
		t.Error("expected both bullets")
	}
}

func TestSaveInitsGit(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	st.Save("Test", []string{"bullet"})

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Error("expected .git directory to be created")
	}
}

func TestReadExisting(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	st.Save("SQL", []string{"use indexes"})

	content, err := st.Read("SQL")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(content, "use indexes") {
		t.Error("expected content")
	}
}

func TestReadNotFound(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	_, err := st.Read("nonexistent")
	if err == nil {
		t.Error("expected error for missing topic")
	}
}

func TestReadRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	for _, bad := range []string{"../etc/passwd", "foo/bar", "test\\path"} {
		_, err := st.Read(bad)
		if err == nil {
			t.Errorf("expected error for topic %q", bad)
		}
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	st.Save("Go", []string{"bullet"})
	st.Save("SQL", []string{"bullet"})

	topics, err := st.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}
}

func TestListEmpty(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	topics, err := st.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(topics) != 0 {
		t.Fatalf("expected 0 topics, got %d", len(topics))
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	st.Save("Go", []string{"goroutines are lightweight"})
	st.Save("SQL", []string{"indexes improve queries"})

	results, err := st.Search("goroutine")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].Topic != "go" {
		t.Errorf("expected topic 'go', got %q", results[0].Topic)
	}
}

func TestSearchNoResults(t *testing.T) {
	dir := t.TempDir()
	st := New(dir)

	st.Save("Go", []string{"goroutines"})

	results, err := st.Search("kubernetes")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSlugify(t *testing.T) {
	st := New("")
	tests := []struct {
		input string
		want  string
	}{
		{"Go Basics", "go-basics"},
		{"SQL & Databases", "sql-databases"},
		{"  spaces  ", "spaces"},
		{"Hello World!", "hello-world"},
	}
	for _, tt := range tests {
		got := st.Slug(tt.input)
		if got != tt.want {
			t.Errorf("Slug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
