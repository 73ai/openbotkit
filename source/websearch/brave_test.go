package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const braveFixtureHTML = `<!DOCTYPE html>
<html>
<body>
<div data-type="web">
  <a href="https://example.com/brave1">Brave Result One</a>
  <div class="snippet-content">Snippet for brave result one</div>
</div>
<div data-type="web">
  <a href="https://example.com/brave2">Brave Result Two</a>
  <div class="snippet-content">Snippet for brave result two</div>
</div>
<div data-type="web">
  <a href="https://example.com/brave3">Brave Result Three</a>
  <p class="snippet-description">Snippet for brave result three</p>
</div>
</body>
</html>`

func TestBraveSearchNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, braveFixtureHTML)
	}))
	defer srv.Close()

	b := &Brave{client: srv.Client(), baseURL: srv.URL}
	results, err := b.Search(context.Background(), "test query", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Brave Result One" {
		t.Errorf("expected 'Brave Result One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/brave1" {
		t.Errorf("expected brave1 URL, got %q", results[0].URL)
	}
	if results[0].Snippet != "Snippet for brave result one" {
		t.Errorf("expected snippet, got %q", results[0].Snippet)
	}
	if results[0].Source != "brave" {
		t.Errorf("expected source 'brave', got %q", results[0].Source)
	}
	if results[2].Snippet != "Snippet for brave result three" {
		t.Errorf("expected fallback snippet, got %q", results[2].Snippet)
	}
}

func TestBraveSearchEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><div class="no-results">No results</div></body></html>`)
	}))
	defer srv.Close()

	b := &Brave{client: srv.Client(), baseURL: srv.URL}
	results, err := b.Search(context.Background(), "asdfqwerzxcv", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestBraveSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	b := &Brave{client: srv.Client(), baseURL: srv.URL}
	_, err := b.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestBraveSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, braveFixtureHTML)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	b := &Brave{client: srv.Client(), baseURL: srv.URL}
	_, err := b.Search(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
}
