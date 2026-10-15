package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Health(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	resp, err := c.Health()
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected ok, got %q", resp["status"])
	}
}

func TestClient_DB(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/db/gmail" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DBResponse{
			Columns: []string{"name", "age"},
			Rows:    [][]string{{"alice", "30"}},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	resp, err := c.DB("gmail", "SELECT name, age FROM test")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if len(resp.Columns) != 2 || resp.Columns[0] != "name" {
		t.Fatalf("unexpected columns: %v", resp.Columns)
	}
	if len(resp.Rows) != 1 || resp.Rows[0][0] != "alice" {
		t.Fatalf("unexpected rows: %v", resp.Rows)
	}
}

func TestClient_DB_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	_, err := c.DB("gmail", "SELECT 1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_BasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	// Without auth
	c := NewClient(srv.URL, "", "")
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected error without auth")
	}

	// With auth
	c = NewClient(srv.URL, "user", "pass")
	resp, err := c.Health()
	if err != nil {
		t.Fatalf("health with auth: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected ok, got %q", resp["status"])
	}
}

func TestClient_MemoryAdd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int64{"id": 42})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	id, err := c.MemoryAdd("test fact", "preference", "manual")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}
}

func TestClient_MemoryList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]MemoryItem{
			{ID: 1, Content: "fact1", Category: "preference"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	items, err := c.MemoryList("")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Content != "fact1" {
		t.Fatalf("unexpected items: %v", items)
	}
}

func TestClient_MemoryDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "")
	if err := c.MemoryDelete(1); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestClient_WaitWhatsAppAuth(t *testing.T) {
	cases := []struct {
		name    string
		status  map[string]any
		wantErr string
	}{
		{"linked", map[string]any{"authenticated": true, "error": ""}, ""},
		{"failed", map[string]any{"authenticated": false, "error": "whatsapp login failed (err-client-outdated)"}, "err-client-outdated"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/auth/whatsapp/api/qr" || r.URL.Query().Get("account") != "work" {
				t.Errorf("%s: unexpected request %s", tc.name, r.URL)
			}
			json.NewEncoder(w).Encode(tc.status)
		}))
		err := NewClient(srv.URL, "", "").WaitWhatsAppAuth("work")
		srv.Close()
		if tc.wantErr == "" && err != nil {
			t.Errorf("%s: %v", tc.name, err)
		}
		if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.wantErr)
		}
	}
}
