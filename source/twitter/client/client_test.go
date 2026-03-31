package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testSession() *Session {
	return &Session{
		AuthToken: "test-auth-token",
		CSRFToken: "abcdef0123456789abcdef0123456789",
	}
}

func testEndpoints(queryID string) map[string]Endpoint {
	return map[string]Endpoint{
		"TestOp": {QueryID: queryID, Method: "GET"},
	}
}

func allTestEndpoints() map[string]Endpoint {
	return map[string]Endpoint{
		"HomeTimeline":       {QueryID: "ht1", Method: "GET"},
		"HomeLatestTimeline": {QueryID: "hlt1", Method: "GET"},
		"TweetDetail":        {QueryID: "td1", Method: "GET"},
		"SearchTimeline":     {QueryID: "st1", Method: "GET"},
		"UserByScreenName":   {QueryID: "usn1", Method: "GET"},
		"UserTweets":         {QueryID: "ut1", Method: "GET"},
		"CreateTweet":        {QueryID: "ct1", Method: "POST"},
		"CreateRetweet":      {QueryID: "cr1", Method: "POST"},
		"FavoriteTweet":      {QueryID: "ft1", Method: "POST"},
		"Notifications":      {QueryID: "n1", Method: "GET"},
	}
}

func TestNewClient_NilSession(t *testing.T) {
	_, err := NewClient(nil, "")
	if err == nil {
		t.Fatal("expected error for nil session")
	}
}

func TestNewClient_EmptyAuthToken(t *testing.T) {
	_, err := NewClient(&Session{}, "")
	if err == nil {
		t.Fatal("expected error for empty auth_token")
	}
}

func TestDoRequest_Headers(t *testing.T) {
	var capturedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	session := testSession()
	c := NewClientWithHTTP(session, srv.Client(), testEndpoints("qid1"), srv.URL)

	ctx := context.Background()
	_, err := c.doRequest(ctx, "GET", srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}

	if got := capturedHeaders.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
		t.Errorf("Authorization header = %q, want Bearer prefix", got)
	}
	if got := capturedHeaders.Get("X-Csrf-Token"); got != session.CSRFToken {
		t.Errorf("X-Csrf-Token = %q, want %q", got, session.CSRFToken)
	}
	if got := capturedHeaders.Get("Cookie"); got == "" {
		t.Error("Cookie header should not be empty")
	}
	if got := capturedHeaders.Get("X-Client-Transaction-Id"); got == "" {
		t.Error("X-Client-Transaction-Id should not be empty")
	}
}

func TestDoRequest_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":[{"message":"forbidden"}]}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), testEndpoints("qid1"), srv.URL)

	_, err := c.doRequest(context.Background(), "GET", srv.URL+"/test", nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestDoRequest_ValidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"key":"value"}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), testEndpoints("qid1"), srv.URL)

	raw, err := c.doRequest(context.Background(), "GET", srv.URL+"/test", nil)
	if err != nil {
		t.Fatalf("doRequest: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	data := result["data"].(map[string]any)
	if data["key"] != "value" {
		t.Errorf("data.key = %v, want value", data["key"])
	}
}

func TestBuildHeaders(t *testing.T) {
	session := testSession()
	c := NewClientWithHTTP(session, http.DefaultClient, testEndpoints("qid1"), "")

	headers := c.buildHeaders()

	if got := headers.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
		t.Errorf("Authorization = %q, want Bearer prefix", got)
	}
	if got := headers.Get("X-Csrf-Token"); got != session.CSRFToken {
		t.Errorf("X-Csrf-Token = %q, want %q", got, session.CSRFToken)
	}
	cookie := headers.Get("Cookie")
	if !strings.Contains(cookie, "auth_token=test-auth-token") {
		t.Errorf("Cookie missing auth_token: %q", cookie)
	}
	if !strings.Contains(cookie, "ct0="+session.CSRFToken) {
		t.Errorf("Cookie missing ct0: %q", cookie)
	}
}

func TestDoRequest_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), testEndpoints("qid1"), srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.doRequest(ctx, "GET", srv.URL+"/test", nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
