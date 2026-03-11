package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const yandexFixtureHTML = `<!DOCTYPE html>
<html>
<body>
<li class="serp-item">
  <a href="https://example.com/yandex1">Yandex Result One</a>
  <div class="text-container">Snippet for yandex result one</div>
</li>
<li class="serp-item">
  <a href="https://example.com/yandex2">Yandex Result Two</a>
  <div class="text-container">Snippet for yandex result two</div>
</li>
<li class="serp-item">
  <a href="https://example.com/yandex3">Yandex Result Three</a>
  <div class="text-container">Snippet for yandex result three</div>
</li>
</body>
</html>`

func TestYandexSearchNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yandexFixtureHTML)
	}))
	defer srv.Close()

	y := &Yandex{client: srv.Client(), baseURL: srv.URL}
	results, err := y.Search(context.Background(), "test query", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Yandex Result One" {
		t.Errorf("expected 'Yandex Result One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/yandex1" {
		t.Errorf("expected yandex1 URL, got %q", results[0].URL)
	}
	if results[0].Snippet != "Snippet for yandex result one" {
		t.Errorf("expected snippet, got %q", results[0].Snippet)
	}
	if results[0].Source != "yandex" {
		t.Errorf("expected source 'yandex', got %q", results[0].Source)
	}
}

func TestYandexSearchEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	y := &Yandex{client: srv.Client(), baseURL: srv.URL}
	results, err := y.Search(context.Background(), "asdfqwerzxcv", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestYandexSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	y := &Yandex{client: srv.Client(), baseURL: srv.URL}
	_, err := y.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestYandexSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, yandexFixtureHTML)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	y := &Yandex{client: srv.Client(), baseURL: srv.URL}
	_, err := y.Search(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
}
