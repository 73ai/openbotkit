package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearch_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[]}}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.Search(context.Background(), "golang", 20, "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if !strings.Contains(capturedPath, "/st1/SearchTimeline") {
		t.Errorf("path = %q, want to contain /st1/SearchTimeline", capturedPath)
	}
	if !strings.Contains(capturedQuery, "golang") {
		t.Error("query should contain search term")
	}
}

func TestSearch_WithCursor(t *testing.T) {
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"search_by_raw_query":{"search_timeline":{"timeline":{"instructions":[]}}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.Search(context.Background(), "test", 10, "page2-cursor")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if !strings.Contains(capturedQuery, "page2-cursor") {
		t.Error("query should contain cursor when provided")
	}
}
