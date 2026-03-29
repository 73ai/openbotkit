package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateTweet_PostsCorrectBody(t *testing.T) {
	var capturedBody map[string]any
	var capturedMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"create_tweet":{"tweet_results":{}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)

	_, err := c.CreateTweet(context.Background(), "hello from test", "")
	if err != nil {
		t.Fatalf("CreateTweet: %v", err)
	}

	if capturedMethod != "POST" {
		t.Errorf("method = %q, want POST", capturedMethod)
	}

	vars, ok := capturedBody["variables"].(map[string]any)
	if !ok {
		t.Fatal("expected variables in body")
	}
	if vars["tweet_text"] != "hello from test" {
		t.Errorf("tweet_text = %q, want 'hello from test'", vars["tweet_text"])
	}
}

func TestCreateTweet_WithReply(t *testing.T) {
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"create_tweet":{"tweet_results":{}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)

	_, err := c.CreateTweet(context.Background(), "reply text", "12345")
	if err != nil {
		t.Fatalf("CreateTweet reply: %v", err)
	}

	vars := capturedBody["variables"].(map[string]any)
	reply, ok := vars["reply"].(map[string]any)
	if !ok {
		t.Fatal("expected reply field in variables")
	}
	if reply["in_reply_to_tweet_id"] != "12345" {
		t.Errorf("in_reply_to_tweet_id = %v, want 12345", reply["in_reply_to_tweet_id"])
	}
}

func TestFavoriteTweet_PostsCorrectly(t *testing.T) {
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"favorite_tweet":"Done"}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)

	_, err := c.FavoriteTweet(context.Background(), "99999")
	if err != nil {
		t.Fatalf("FavoriteTweet: %v", err)
	}

	vars := capturedBody["variables"].(map[string]any)
	if vars["tweet_id"] != "99999" {
		t.Errorf("tweet_id = %v, want 99999", vars["tweet_id"])
	}
}

func TestCreateRetweet_PostsCorrectly(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"create_retweet":{}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)

	_, err := c.CreateRetweet(context.Background(), "55555")
	if err != nil {
		t.Fatalf("CreateRetweet: %v", err)
	}

	if !strings.Contains(capturedPath, "/cr1/CreateRetweet") {
		t.Errorf("path = %q, want to contain /cr1/CreateRetweet", capturedPath)
	}
}

func TestGetTweet_CallsCorrectEndpoint(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"tweetResult":{"result":{}}}}`))
	}))
	defer srv.Close()

	c := NewClientWithHTTP(testSession(), srv.Client(), allTestEndpoints(), srv.URL)

	_, err := c.GetTweet(context.Background(), "77777")
	if err != nil {
		t.Fatalf("GetTweet: %v", err)
	}

	if !strings.Contains(capturedPath, "/td1/TweetDetail") {
		t.Errorf("path = %q, want to contain /td1/TweetDetail", capturedPath)
	}
}

func TestGraphqlPOST_UnknownOperation(t *testing.T) {
	c := NewClientWithHTTP(testSession(), http.DefaultClient, map[string]Endpoint{}, "")

	_, err := c.graphqlPOST(context.Background(), "NonExistentOp", nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
	if !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("error = %q, want 'unknown operation'", err.Error())
	}
}
