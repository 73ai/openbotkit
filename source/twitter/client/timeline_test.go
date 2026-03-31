package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeTimeline_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"home":{"home_timeline_urt":{"instructions":[]}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.HomeTimeline(context.Background(), 20, "")
	if err != nil {
		t.Fatalf("HomeTimeline: %v", err)
	}

	if !strings.Contains(capturedPath, "/ht1/HomeTimeline") {
		t.Errorf("path = %q, want to contain /ht1/HomeTimeline", capturedPath)
	}
	if !strings.Contains(capturedQuery, "variables") {
		t.Error("query should contain variables parameter")
	}
	if !strings.Contains(capturedQuery, "features") {
		t.Error("query should contain features parameter")
	}
}

func TestHomeTimeline_WithCursor(t *testing.T) {
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"home":{"home_timeline_urt":{"instructions":[]}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.HomeTimeline(context.Background(), 20, "cursor-abc")
	if err != nil {
		t.Fatalf("HomeTimeline: %v", err)
	}

	if !strings.Contains(capturedQuery, "cursor") {
		t.Error("query should contain cursor when provided")
	}
}

func TestHomeLatestTimeline_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"home":{"home_timeline_urt":{"instructions":[]}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.HomeLatestTimeline(context.Background(), 20, "")
	if err != nil {
		t.Fatalf("HomeLatestTimeline: %v", err)
	}

	if !strings.Contains(capturedPath, "/hlt1/HomeLatestTimeline") {
		t.Errorf("path = %q, want to contain /hlt1/HomeLatestTimeline", capturedPath)
	}
}

func TestGraphqlGET_UnknownOperation(t *testing.T) {
	c := NewClientWithHTTP(testSession(), http.DefaultClient, map[string]Endpoint{}, "")

	_, err := c.graphqlGET(context.Background(), "NonExistentOp", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
	if !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("error = %q, want 'unknown operation'", err.Error())
	}
}
