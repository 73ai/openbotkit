package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotifications_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"viewer":{"notifications_timeline":{"timeline":{"instructions":[]}}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.Notifications(context.Background(), 20, "")
	if err != nil {
		t.Fatalf("Notifications: %v", err)
	}

	if !strings.Contains(capturedPath, "/n1/Notifications") {
		t.Errorf("path = %q, want to contain /n1/Notifications", capturedPath)
	}
}

func TestNotifications_WithCursor(t *testing.T) {
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"viewer":{"notifications_timeline":{"timeline":{"instructions":[]}}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.Notifications(context.Background(), 10, "notif-cursor")
	if err != nil {
		t.Fatalf("Notifications: %v", err)
	}

	if !strings.Contains(capturedQuery, "notif-cursor") {
		t.Error("query should contain cursor when provided")
	}
}
