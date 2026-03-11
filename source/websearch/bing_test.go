package websearch

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

const bingTestHTML = `<html><body>
<ol id="b_results">
<li class="b_algo">
<h2><a href="https://example.com/go">Go Programming</a></h2>
<p>Go is an open source programming language</p>
</li>
<li class="b_algo">
<h2><a href="https://example.com/rust">Rust Language</a></h2>
<p>A systems programming language</p>
</li>
</ol>
</body></html>`

func TestBingSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(bingTestHTML))
	}))
	defer srv.Close()

	b := &Bing{client: srv.Client(), baseURL: srv.URL}
	results, err := b.Search(context.Background(), "golang", SearchOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "Go Programming" {
		t.Errorf("expected 'Go Programming', got %q", results[0].Title)
	}
	if results[0].Source != "bing" {
		t.Errorf("expected source 'bing', got %q", results[0].Source)
	}
}

func TestBingMaxResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(bingTestHTML))
	}))
	defer srv.Close()

	b := &Bing{client: srv.Client(), baseURL: srv.URL}
	results, err := b.Search(context.Background(), "test", SearchOptions{MaxResults: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestUnwrapBingURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain URL", "https://example.com", "https://example.com"},
		{"non-bing redirect", "https://other.com/page", "https://other.com/page"},
		{"bing redirect with base64 encoded URL",
			"https://www.bing.com/ck/a?u=a1" + base64.RawURLEncoding.EncodeToString([]byte("https://decoded.example.com")),
			"https://decoded.example.com"},
		{"bing redirect too short u param", "https://www.bing.com/ck/a?u=ab", "https://www.bing.com/ck/a?u=ab"},
		{"bing redirect empty u param", "https://www.bing.com/ck/a?u=", "https://www.bing.com/ck/a?u="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapBingURL(tt.input)
			if got != tt.expected {
				t.Errorf("unwrapBingURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsBingAdURL(t *testing.T) {
	if !isBingAdURL("https://www.bing.com/aclick?ld=...") {
		t.Error("expected ad URL to be detected")
	}
	if isBingAdURL("https://example.com") {
		t.Error("expected non-ad URL to pass")
	}
}

func TestBingFilterAds(t *testing.T) {
	html := `<html><body><ol id="b_results">
<li class="b_algo">
<h2><a href="https://www.bing.com/aclick?ld=foo">Ad</a></h2>
<p>An ad result</p>
</li>
<li class="b_algo">
<h2><a href="https://real.com">Real Result</a></h2>
<p>A real result</p>
</li>
</ol></body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer srv.Close()

	b := &Bing{client: srv.Client(), baseURL: srv.URL}
	results, err := b.Search(context.Background(), "test", SearchOptions{MaxResults: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (ads filtered), got %d", len(results))
	}
	if results[0].URL != "https://real.com" {
		t.Errorf("expected real result, got %q", results[0].URL)
	}
}
