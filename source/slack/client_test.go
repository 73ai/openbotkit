package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func testServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("xoxp-test-token", "")
	c.baseURL = srv.URL
	return c
}

func testBrowserServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("xoxc-browser-token", "xoxd-cookie")
	c.baseURL = srv.URL
	return c
}

func TestClient_StandardAuth(t *testing.T) {
	var gotAuth string
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "123"})
	})

	_, err := c.PostMessage(context.Background(), "C123", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer xoxp-test-token" {
		t.Errorf("auth header = %q", gotAuth)
	}
}

func TestClient_BrowserAuth(t *testing.T) {
	var gotCookie, gotAuth string
	var gotToken string
	c := testBrowserServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")
		r.ParseForm()
		gotToken = r.FormValue("token")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "123"})
	})

	_, err := c.PostMessage(context.Background(), "C123", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Errorf("browser auth should not have Authorization header, got %q", gotAuth)
	}
	if gotCookie != "d=xoxd-cookie" {
		t.Errorf("cookie = %q", gotCookie)
	}
	if gotToken != "xoxc-browser-token" {
		t.Errorf("form token = %q", gotToken)
	}
}

func TestClient_ErrorParsing(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "channel_not_found"})
	})

	_, err := c.ConversationsHistory(context.Background(), "C999", HistoryOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "slack API conversations.history: channel_not_found" {
		t.Errorf("error = %q", got)
	}
}

func TestClient_RateLimitRetry(t *testing.T) {
	var calls atomic.Int32
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "messages": []any{}})
	})

	msgs, err := c.ConversationsHistory(context.Background(), "C123", HistoryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty messages, got %d", len(msgs))
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestClient_RateLimitExhausted(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(429)
	})

	_, err := c.ConversationsHistory(context.Background(), "C123", HistoryOptions{})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
}

func TestClient_SearchMessages(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": map[string]any{
				"total": 1,
				"page":  1,
				"pages": 1,
				"matches": []map[string]any{
					{"ts": "1234", "text": "found it", "user": "U123"},
				},
			},
		})
	})

	result, err := c.SearchMessages(context.Background(), "test query", SearchOptions{Count: 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d", result.Total)
	}
	if len(result.Messages) != 1 || result.Messages[0].Text != "found it" {
		t.Errorf("messages = %+v", result.Messages)
	}
}

func TestClient_AuthTest(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"team_id": "T123",
			"team":    "TestTeam",
			"user_id": "U456",
		})
	})

	teamID, teamName, userID, err := c.AuthTest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if teamID != "T123" || teamName != "TestTeam" || userID != "U456" {
		t.Errorf("got team=%q name=%q user=%q", teamID, teamName, userID)
	}
}

func TestClient_PostMessage(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.FormValue("channel") != "C123" || r.FormValue("text") != "hello" {
			t.Errorf("wrong params: channel=%q text=%q", r.FormValue("channel"), r.FormValue("text"))
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "999.999"})
	})

	ts, err := c.PostMessage(context.Background(), "C123", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if ts != "999.999" {
		t.Errorf("ts = %q", ts)
	}
}

func TestClient_AddReaction(t *testing.T) {
	c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.FormValue("name") != "thumbsup" {
			t.Errorf("emoji = %q", r.FormValue("name"))
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	err := c.AddReaction(context.Background(), "C123", "1234.5678", "thumbsup")
	if err != nil {
		t.Fatal(err)
	}
}
