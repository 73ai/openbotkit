package websearch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const yahooFixtureHTML = `<!DOCTYPE html>
<html>
<body>
<div class="algo">
  <a href="/RU=https%3A%2F%2Fexample.com%2Fyahoo1/RK=0/RS=0">Yahoo Result One</a>
  <div class="compText">Snippet for yahoo result one</div>
</div>
<div class="algo">
  <a href="/RU=https%3A%2F%2Fexample.com%2Fyahoo2/RK=0/RS=0">Yahoo Result Two</a>
  <div class="compText">Snippet for yahoo result two</div>
</div>
<div class="algo">
  <a href="/RU=https%3A%2F%2Fexample.com%2Fyahoo3/RK=0/RS=0">Yahoo Result Three</a>
  <div class="compText">Snippet for yahoo result three</div>
</div>
</body>
</html>`

func TestYahooSearchNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, yahooFixtureHTML)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL}
	results, err := y.Search(context.Background(), "test query", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Yahoo Result One" {
		t.Errorf("expected 'Yahoo Result One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/yahoo1" {
		t.Errorf("expected unwrapped URL, got %q", results[0].URL)
	}
	if results[0].Snippet != "Snippet for yahoo result one" {
		t.Errorf("expected snippet, got %q", results[0].Snippet)
	}
	if results[0].Source != "yahoo" {
		t.Errorf("expected source 'yahoo', got %q", results[0].Source)
	}
}

func TestYahooSearchAdFiltering(t *testing.T) {
	html := `<html><body>
<div class="algo">
  <a href="https://r.search.yahoo.com/aclick?some_ad_url">Ad Result</a>
  <div class="compText">Ad snippet</div>
</div>
<div class="algo">
  <a href="https://www.bing.com/aclick?some_other_ad">Bing Ad</a>
  <div class="compText">Bing ad snippet</div>
</div>
<div class="algo">
  <a href="/RU=https%3A%2F%2Fexample.com%2Freal/RK=0/RS=0">Real Result</a>
  <div class="compText">Real snippet</div>
</div>
</body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, html)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL}
	results, err := y.Search(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (ads filtered), got %d", len(results))
	}
	if results[0].Title != "Real Result" {
		t.Errorf("expected 'Real Result', got %q", results[0].Title)
	}
}

func TestUnwrapYahooURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "standard RU pattern",
			raw:  "/RU=https%3A%2F%2Fexample.com%2Fpage/RK=0/RS=0",
			want: "https://example.com/page",
		},
		{
			name: "encoded query params",
			raw:  "/RU=https%3A%2F%2Fexample.com%2Fsearch%3Fq%3Dtest/RK=2/RS=abc",
			want: "https://example.com/search?q=test",
		},
		{
			name: "no RU pattern",
			raw:  "https://example.com/direct",
			want: "https://example.com/direct",
		},
		{
			name: "no RK pattern",
			raw:  "/RU=https%3A%2F%2Fexample.com/nope",
			want: "/RU=https%3A%2F%2Fexample.com/nope",
		},
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapYahooURL(tt.raw)
			if got != tt.want {
				t.Errorf("unwrapYahooURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestYahooSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL}
	_, err := y.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestYahooSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, yahooFixtureHTML)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL}
	_, err := y.Search(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
}

func TestYahooSearchTimeLimitParam(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	y := &Yahoo{client: srv.Client(), baseURL: srv.URL}
	_, _ = y.Search(context.Background(), "test", SearchOptions{TimeLimit: "w"})

	if gotQuery.Get("btf") != "w" {
		t.Errorf("expected btf=w, got %q", gotQuery.Get("btf"))
	}
}
