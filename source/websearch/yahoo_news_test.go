package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const yahooNewsFixtureHTML = `<!DOCTYPE html>
<html>
<body>
<div class="NewsArticle">
  <a href="/RU=https%3A%2F%2Fexample.com%2Fnews1/RK=0/RS=0">Yahoo News One</a>
  <p>Excerpt for news one</p>
  <span class="s-time">2 hours ago</span>
  <span class="s-source">Example News</span>
  <img src="https://img.com/news1.jpg">
</div>
<div class="NewsArticle">
  <a href="/RU=https%3A%2F%2Fexample.com%2Fnews2/RK=0/RS=0">Yahoo News Two</a>
  <p>Excerpt for news two</p>
  <span class="s-time">5 hours ago</span>
  <span class="s-source">Other News</span>
</div>
</body>
</html>`

func TestYahooNewsNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yahooNewsFixtureHTML)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL, newsURL: srv.URL}
	results, err := y.News(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Yahoo News One" {
		t.Errorf("expected 'Yahoo News One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/news1" {
		t.Errorf("expected unwrapped URL, got %q", results[0].URL)
	}
	if results[0].Date != "2 hours ago" {
		t.Errorf("expected '2 hours ago', got %q", results[0].Date)
	}
	if results[0].Image != "https://img.com/news1.jpg" {
		t.Errorf("expected image URL, got %q", results[0].Image)
	}
	if results[0].Source != "yahoo" {
		t.Errorf("expected source 'yahoo', got %q", results[0].Source)
	}
}

func TestYahooNewsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL, newsURL: srv.URL}
	results, err := y.News(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestYahooNewsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL, newsURL: srv.URL}
	_, err := y.News(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}
