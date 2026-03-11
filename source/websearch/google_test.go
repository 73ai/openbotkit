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

const googleFixtureHTML = `<!DOCTYPE html>
<html>
<body>
<div class="g">
  <a href="/url?q=https://example.com/google1&sa=U"><h3>Google Result One</h3></a>
  <div class="VwiC3b">Snippet for google result one</div>
</div>
<div class="g">
  <a href="/url?q=https://example.com/google2&sa=U"><h3>Google Result Two</h3></a>
  <div class="VwiC3b">Snippet for google result two</div>
</div>
<div class="g">
  <a href="https://example.com/google3"><h3>Google Result Three</h3></a>
  <div class="VwiC3b">Snippet for google result three</div>
</div>
</body>
</html>`

func TestGoogleSearchNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, googleFixtureHTML)
	}))
	defer srv.Close()

	g := &Google{client: srv.Client(), baseURL: srv.URL}
	results, err := g.Search(context.Background(), "test query", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Title != "Google Result One" {
		t.Errorf("expected 'Google Result One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/google1" {
		t.Errorf("expected unwrapped URL, got %q", results[0].URL)
	}
	if results[0].Snippet != "Snippet for google result one" {
		t.Errorf("expected snippet, got %q", results[0].Snippet)
	}
	if results[0].Source != "google" {
		t.Errorf("expected source 'google', got %q", results[0].Source)
	}
	if results[2].URL != "https://example.com/google3" {
		t.Errorf("expected direct URL, got %q", results[2].URL)
	}
}

func TestUnwrapGoogleURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "standard /url?q= pattern",
			raw:  "/url?q=https://example.com/page&sa=U&ved=abc",
			want: "https://example.com/page",
		},
		{
			name: "direct URL",
			raw:  "https://example.com/direct",
			want: "https://example.com/direct",
		},
		{
			name: "/url without q param",
			raw:  "/url?sa=U&ved=abc",
			want: "/url?sa=U&ved=abc",
		},
		{
			name: "empty string",
			raw:  "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapGoogleURL(tt.raw)
			if got != tt.want {
				t.Errorf("unwrapGoogleURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestGoogleSearchTimeLimitParam(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer srv.Close()

	g := &Google{client: srv.Client(), baseURL: srv.URL}
	_, _ = g.Search(context.Background(), "test", SearchOptions{TimeLimit: "w"})

	if gotQuery.Get("tbs") != "qdr:w" {
		t.Errorf("expected tbs=qdr:w, got %q", gotQuery.Get("tbs"))
	}
}

func TestGoogleSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	g := &Google{client: srv.Client(), baseURL: srv.URL}
	_, err := g.Search(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGoogleSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, googleFixtureHTML)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	g := &Google{client: srv.Client(), baseURL: srv.URL}
	_, err := g.Search(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
}
