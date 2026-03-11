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

func TestDDGNewsNormal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "news.js") {
			fmt.Fprint(w, `{"results":[
				{"title":"News One","url":"https://example.com/news1","excerpt":"Excerpt one","source":"Example","date":1700000000,"image":"https://img.com/1.jpg"},
				{"title":"News Two","url":"https://example.com/news2","excerpt":"Excerpt two","source":"Example","date":1700100000,"image":""}
			]}`)
			return
		}
		fmt.Fprint(w, `<html><script>vqd="test-vqd-token"</script></html>`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{
		client:  srv.Client(),
		baseURL: srv.URL,
		vqdURL:  srv.URL + "/",
		newsURL: srv.URL + "/news.js",
	}

	results, err := d.News(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "News One" {
		t.Errorf("expected 'News One', got %q", results[0].Title)
	}
	if results[0].URL != "https://example.com/news1" {
		t.Errorf("expected news1 URL, got %q", results[0].URL)
	}
	if results[0].Snippet != "Excerpt one" {
		t.Errorf("expected excerpt, got %q", results[0].Snippet)
	}
	if results[0].Source != "duckduckgo" {
		t.Errorf("expected source 'duckduckgo', got %q", results[0].Source)
	}
	if results[0].Image != "https://img.com/1.jpg" {
		t.Errorf("expected image URL, got %q", results[0].Image)
	}
	if results[0].Date != "1700000000" {
		t.Errorf("expected date timestamp, got %q", results[0].Date)
	}
}

func TestDDGNewsVQDFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html>no token here</html>`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{
		client:  srv.Client(),
		baseURL: srv.URL,
		vqdURL:  srv.URL + "/",
		newsURL: srv.URL + "/news.js",
	}

	_, err := d.News(context.Background(), "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected error when VQD token not found")
	}
	if !strings.Contains(err.Error(), "vqd") {
		t.Errorf("error should mention vqd, got: %v", err)
	}
}

func TestDDGNewsNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "news.js") {
			fmt.Fprint(w, `{"results":[]}`)
			return
		}
		fmt.Fprint(w, `<html><script>vqd="token123"</script></html>`)
	}))
	defer srv.Close()

	d := &DuckDuckGo{
		client:  srv.Client(),
		baseURL: srv.URL,
		vqdURL:  srv.URL + "/",
		newsURL: srv.URL + "/news.js",
	}

	results, err := d.News(context.Background(), "test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestDDGNewsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, `<html><script>vqd="token"</script></html>`)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	d := &DuckDuckGo{
		client:  srv.Client(),
		baseURL: srv.URL,
		vqdURL:  srv.URL + "/",
		newsURL: srv.URL + "/news.js",
	}

	_, err := d.News(ctx, "test", SearchOptions{})
	if err == nil {
		t.Fatal("expected deadline exceeded error")
	}
}

func TestVQDExtraction(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantVQD string
		wantErr bool
	}{
		{
			name:    "standard vqd attribute",
			html:    `<html><script>vqd="abc123"</script></html>`,
			wantVQD: "abc123",
		},
		{
			name:    "vqd with hyphens",
			html:    `<html>vqd="4-123456789-abcdef"</html>`,
			wantVQD: "4-123456789-abcdef",
		},
		{
			name:    "no vqd token",
			html:    `<html><body>no token</body></html>`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tt.html)
			}))
			defer srv.Close()

			d := &DuckDuckGo{
				client: srv.Client(),
				vqdURL: srv.URL + "/",
			}
			vqd, err := d.fetchVQD(context.Background(), "test")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if vqd != tt.wantVQD {
				t.Errorf("got vqd %q, want %q", vqd, tt.wantVQD)
			}
		})
	}
}
