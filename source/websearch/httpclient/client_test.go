package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSetsHeaders(t *testing.T) {
	var gotUA, gotAccept, gotLang string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		gotLang = r.Header.Get("Accept-Language")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(nil, WithBrowserClient(srv.Client()))
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotUA == "" {
		t.Error("expected User-Agent to be set")
	}
	if gotAccept == "" {
		t.Error("expected Accept to be set")
	}
	if gotLang == "" {
		t.Error("expected Accept-Language to be set")
	}
}

func TestClientPreservesExplicitHeaders(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := New(nil, WithBrowserClient(srv.Client()))
	req, _ := http.NewRequest("GET", srv.URL+"/test", nil)
	req.Header.Set("User-Agent", "custom-agent")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotUA != "custom-agent" {
		t.Errorf("expected custom UA preserved, got %q", gotUA)
	}
}

func TestClientProfileConsistent(t *testing.T) {
	c := New(nil, WithBrowserClient(http.DefaultClient))
	if c.profile.UserAgent == "" {
		t.Error("profile should have a User-Agent")
	}
}
