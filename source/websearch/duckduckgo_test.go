package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const ddgFixtureHTML = `<!DOCTYPE html>
<html>
<body>
<div class="result">
  <a class="result__a" href="https://example.com/page1">Result One</a>
  <a class="result__snippet">Snippet for result one</a>
</div>
<div class="result">
  <a class="result__a" href="https://example.com/page2">Result Two</a>
  <a class="result__snippet">Snippet for result two</a>
</div>
<div class="result">
  <a class="result__a" href="https://example.com/page3">Result Three</a>
  <a class="result__snippet">Snippet for result three</a>
</div>
</body>
</html>`

func TestDDGSearchNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, ddgFixtureHTML)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	results, err := d.Search(context.Background(), "test query", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Result One" {
		t.Errorf("expected 'Result One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/page1" {
		t.Errorf("expected page1 URL, got %q", results[0].URL)
	}
	if results[0].Snippet != "Snippet for result one" {
		t.Errorf("expected snippet, got %q", results[0].Snippet)
	}
	if results[0].Source != "duckduckgo" {
		t.Errorf("expected source 'duckduckgo', got %q", results[0].Source)
	}
}

func TestDDGSearchFilterRedirects(t *testing.T) {
	html := `<html><body>
<div class="result">
  <a class="result__a" href="https://duckduckgo.com/y.js?ad_provider=bingv7aa">Ad Result</a>
  <a class="result__snippet">Ad snippet</a>
</div>
<div class="result">
  <a class="result__a" href="https://example.com/real">Real Result</a>
  <a class="result__snippet">Real snippet</a>
</div>
</body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	results, err := d.Search(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (redirect filtered), got %d", len(results))
	}
	if results[0].Title != "Real Result" {
		t.Errorf("expected 'Real Result', got %q", results[0].Title)
	}
}

func TestDDGSearchEmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><div class="no-results">No results</div></body></html>`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	results, err := d.Search(context.Background(), "asdfqwerzxcv", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestDDGSearchQueryTruncation(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	longQuery := strings.Repeat("a", 600)
	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	_, _ = d.Search(context.Background(), longQuery, SearchOptions{})

	if strings.Contains(received, strings.Repeat("a", 600)) {
		t.Error("query was not truncated")
	}
	if !strings.Contains(received, strings.Repeat("a", 499)) {
		t.Error("query should be truncated to 499 chars")
	}
}

func TestDDGSearchQueryTruncationUTF8(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	// 600 multi-byte runes — truncation must not split a rune.
	multiByteQuery := strings.Repeat("日", 600)
	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	_, _ = d.Search(context.Background(), multiByteQuery, SearchOptions{})

	// Parse the form body and check the query value.
	vals, err := url.ParseQuery(received)
	if err != nil {
		t.Fatalf("invalid form body: %v", err)
	}
	runes := []rune(vals.Get("q"))
	if len(runes) > maxQueryLen {
		t.Errorf("query runes not truncated: got %d runes", len(runes))
	}
}

func TestDDGSearchRegionParam(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	_, _ = d.Search(context.Background(), "test", SearchOptions{Region: "de-de"})

	if !strings.Contains(received, "kl=de-de") {
		t.Errorf("expected kl=de-de in body, got %q", received)
	}
}

func TestDDGSearchTimeLimitParam(t *testing.T) {
	var received string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received = string(body)
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	_, _ = d.Search(context.Background(), "test", SearchOptions{TimeLimit: "w"})

	if !strings.Contains(received, "df=w") {
		t.Errorf("expected df=w in body, got %q", received)
	}
}

func TestDDGSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	_, err := d.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestDDGSearchMalformedHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><div class="result"><a class="result__a" broken`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	results, err := d.Search(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("malformed HTML should not error: %v", err)
	}
	// goquery handles malformed HTML gracefully
	_ = results
}

func TestDDGSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, ddgFixtureHTML)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	d := &DuckDuckGo{client: srv.Client(), baseURL: srv.URL}
	_, err := d.Search(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
}
