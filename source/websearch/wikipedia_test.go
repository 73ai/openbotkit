package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func wikiMockServer(opensearchResp, extractResp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch action {
		case "opensearch":
			fmt.Fprint(w, opensearchResp)
		case "query":
			fmt.Fprint(w, extractResp)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
}

func TestWikiSearchNormal(t *testing.T) {
	srv := wikiMockServer(
		`["Go",["Go (programming language)"],[""],["https://en.wikipedia.org/wiki/Go_(programming_language)"]]`,
		`{"query":{"pages":{"123":{"extract":"Go is a statically typed, compiled programming language designed at Google."}}}}`,
	)
	defer srv.Close()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	results, err := w.Search(context.Background(), "Go", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Go (programming language)" {
		t.Errorf("expected title 'Go (programming language)', got %q", results[0].Title)
	}
	if results[0].URL != "https://en.wikipedia.org/wiki/Go_(programming_language)" {
		t.Errorf("unexpected URL: %q", results[0].URL)
	}
	if !strings.Contains(results[0].Snippet, "statically typed") {
		t.Errorf("expected snippet to contain 'statically typed', got %q", results[0].Snippet)
	}
	if results[0].Source != "wikipedia" {
		t.Errorf("expected source 'wikipedia', got %q", results[0].Source)
	}
}

func TestWikiSearchDisambiguation(t *testing.T) {
	srv := wikiMockServer(
		`["Go",["Go"],[""],["https://en.wikipedia.org/wiki/Go"]]`,
		`{"query":{"pages":{"456":{"extract":"Go may refer to:\nGo (game)\nGo (programming language)"}}}}`,
	)
	defer srv.Close()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	results, err := w.Search(context.Background(), "Go", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for disambiguation, got %d", len(results))
	}
}

func TestWikiSearchNoResults(t *testing.T) {
	srv := wikiMockServer(
		`["asdfqwer",[],[],[]]`,
		`{}`,
	)
	defer srv.Close()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	results, err := w.Search(context.Background(), "asdfqwer", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestWikiSearchExtractTruncation(t *testing.T) {
	longExtract := strings.Repeat("abcdefghij", 50) // 500 chars
	srv := wikiMockServer(
		`["test",["Test"],[""],["https://en.wikipedia.org/wiki/Test"]]`,
		fmt.Sprintf(`{"query":{"pages":{"1":{"extract":"%s"}}}}`, longExtract),
	)
	defer srv.Close()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	results, err := w.Search(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Snippet) > wikiMaxSnippet+10 {
		t.Errorf("snippet not truncated: len=%d", len(results[0].Snippet))
	}
	if !strings.HasSuffix(results[0].Snippet, "...") {
		t.Error("truncated snippet should end with ...")
	}
}

func TestWikiSearchMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{not valid json`)
	}))
	defer srv.Close()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	_, err := w.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestWikiSearchExtractFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch action {
		case "opensearch":
			fmt.Fprint(w, `["test",["Test"],[""],["https://en.wikipedia.org/wiki/Test"]]`)
		case "query":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	_, err := w.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error when extract endpoint returns 500")
	}
}

func TestWikiSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	w := &Wikipedia{client: srv.Client(), searchURL: srv.URL}
	_, err := w.Search(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
