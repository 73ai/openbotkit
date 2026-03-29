package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserByScreenName_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"user":{"result":{}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.UserByScreenName(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("UserByScreenName: %v", err)
	}

	if !strings.Contains(capturedPath, "/usn1/UserByScreenName") {
		t.Errorf("path = %q, want to contain /usn1/UserByScreenName", capturedPath)
	}
	if !strings.Contains(capturedQuery, "testuser") {
		t.Error("query should contain screen name")
	}
}

func TestUserTweets_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"user":{"result":{"timeline_v2":{"timeline":{"instructions":[]}}}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)
	_, err := c.UserTweets(context.Background(), "user123", 20, "")
	if err != nil {
		t.Fatalf("UserTweets: %v", err)
	}

	if !strings.Contains(capturedPath, "/ut1/UserTweets") {
		t.Errorf("path = %q, want to contain /ut1/UserTweets", capturedPath)
	}
}
