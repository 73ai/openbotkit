package websearch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchHTMLToMarkdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>Test Page</title></head><body><h1>Hello</h1><p>World</p></body></html>`)
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{Format: "markdown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", result.Title)
	}
	if !strings.Contains(result.Content, "Hello") {
		t.Errorf("expected content to contain 'Hello', got %q", result.Content)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
}

func TestFetchHTMLToText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><p>Plain text content</p><script>alert('x')</script></body></html>`)
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{Format: "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "Plain text content") {
		t.Errorf("expected plain text content, got %q", result.Content)
	}
}

func TestFetchJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"key":"value","num":42}`)
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "\"key\": \"value\"") {
		t.Errorf("expected pretty-printed JSON, got %q", result.Content)
	}
}

func TestFetchPlainText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Just plain text")
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "Just plain text" {
		t.Errorf("expected 'Just plain text', got %q", result.Content)
	}
}

func TestFetchTruncation(t *testing.T) {
	longContent := strings.Repeat("a", 200)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, longContent)
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{MaxLength: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Truncated {
		t.Error("expected Truncated=true")
	}
	if !strings.Contains(result.Content, "[Content truncated") {
		t.Error("expected truncation marker in content")
	}
}

func TestNormalizeGitHubURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"blob URL", "https://github.com/user/repo/blob/main/file.go",
			"https://raw.githubusercontent.com/user/repo/main/file.go"},
		{"non-blob GitHub URL", "https://github.com/user/repo/issues/1",
			"https://github.com/user/repo/issues/1"},
		{"non-GitHub URL", "https://example.com/github.com/blob/foo",
			"https://example.com/github.com/blob/foo"},
		{"raw URL passthrough", "https://raw.githubusercontent.com/user/repo/main/f.go",
			"https://raw.githubusercontent.com/user/repo/main/f.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeGitHubURL(tt.input)
			if got != tt.want {
				t.Errorf("normalizeGitHubURL(%q):\n  got:  %q\n  want: %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFetchSSRFBlocksLoopback(t *testing.T) {
	ws := &WebSearch{}
	_, err := ws.Fetch(context.Background(), "http://localhost/secret", FetchOptions{})
	if err == nil {
		t.Fatal("expected SSRF error for localhost")
	}
	if !strings.Contains(err.Error(), "private IP") {
		t.Errorf("expected private IP error, got: %v", err)
	}
}

func TestFetchSSRFBlocksPrivateIP(t *testing.T) {
	ws := &WebSearch{}
	_, err := ws.Fetch(context.Background(), "http://127.0.0.1/secret", FetchOptions{})
	if err == nil {
		t.Fatal("expected SSRF error for 127.0.0.1")
	}
}

func TestFetchSSRFBlocksIPv6Loopback(t *testing.T) {
	ws := &WebSearch{}
	_, err := ws.Fetch(context.Background(), "http://[::1]/secret", FetchOptions{})
	if err == nil {
		t.Fatal("expected SSRF error for IPv6 loopback")
	}
}

func TestFetchRejectsNonHTTPScheme(t *testing.T) {
	ws := &WebSearch{skipSSRF: true}
	_, err := ws.Fetch(context.Background(), "file:///etc/passwd", FetchOptions{})
	if err == nil {
		t.Fatal("expected error for file:// scheme")
	}
	if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Errorf("expected scheme error, got: %v", err)
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"::1", true},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := isPrivateIP(ip); got != tt.want {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("HTTP errors should not return Go error: %v", err)
	}
	if result.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", result.StatusCode)
	}
}

func TestFetchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	ws := &WebSearch{skipSSRF: true}
	_, err := ws.Fetch(ctx, srv.URL, FetchOptions{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestFetchDefaultFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Test</h1></body></html>`)
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default format should be markdown, which should contain the heading
	if !strings.Contains(result.Content, "Test") {
		t.Errorf("expected content to contain 'Test', got %q", result.Content)
	}
}

func TestFetchEmptyURL(t *testing.T) {
	ws := &WebSearch{skipSSRF: true}
	_, err := ws.Fetch(context.Background(), "", FetchOptions{})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestFetchResponseBodyCapped(t *testing.T) {
	// Serve a body larger than maxResponseBody (10 MB).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// Write 11 MB (more than the 10 MB cap).
		chunk := strings.Repeat("x", 1<<20) // 1 MB
		for range 11 {
			fmt.Fprint(w, chunk)
		}
	}))
	defer srv.Close()

	ws := &WebSearch{skipSSRF: true}
	result, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{MaxLength: 12 << 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) > maxResponseBody+100 {
		t.Errorf("body not capped: got %d bytes", len(result.Content))
	}
}

func TestFetchCacheIntegration(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "cached content")
	}))
	defer srv.Close()

	db := openTestDB(t)
	ws := &WebSearch{skipSSRF: true, db: db}

	// First fetch — cache miss, server called.
	r1, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 server call, got %d", callCount)
	}
	if r1.Cached {
		t.Error("first fetch should not be cached")
	}
	if r1.Content != "cached content" {
		t.Errorf("expected 'cached content', got %q", r1.Content)
	}

	// Second fetch — cache hit, server NOT called.
	r2, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected server called only once, got %d", callCount)
	}
	if !r2.Cached {
		t.Error("second fetch should be cached")
	}
	if r2.Content != "cached content" {
		t.Errorf("expected cached content, got %q", r2.Content)
	}
}

func TestFetchNoCacheBypass(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "response %d", callCount)
	}))
	defer srv.Close()

	db := openTestDB(t)
	ws := &WebSearch{skipSSRF: true, db: db}

	// First fetch to populate cache.
	_, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	// Second fetch with NoCache — server called again.
	r2, err := ws.Fetch(context.Background(), srv.URL, FetchOptions{NoCache: true})
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls with NoCache, got %d", callCount)
	}
	if r2.Content != "response 2" {
		t.Errorf("expected fresh response, got %q", r2.Content)
	}
}
