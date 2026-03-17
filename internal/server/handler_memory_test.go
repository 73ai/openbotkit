package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/memory"
	"github.com/73ai/openbotkit/store"
)

func testServerWithMemoryDB(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	cfg := config.Default()

	// Ensure the user_memory source directory exists
	memDir := filepath.Join(dir, "user_memory")
	if err := os.MkdirAll(memDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Pre-create the memory DB so the handler can open it
	dsn := cfg.UserMemoryDataDSN()
	db, err := store.Open(store.Config{Driver: "sqlite", DSN: dsn})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := memory.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Close()

	return &Server{cfg: cfg}
}

func TestMemoryList_Empty(t *testing.T) {
	s := testServerWithMemoryDB(t)

	req := httptest.NewRequest("GET", "/api/memory", nil)
	rec := httptest.NewRecorder()

	s.handleMemoryList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var items []memoryItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestMemoryAdd_ThenList(t *testing.T) {
	s := testServerWithMemoryDB(t)

	// Add a memory
	body := `{"content": "likes Go", "category": "preference", "source": "manual"}`
	req := httptest.NewRequest("POST", "/api/memory", strings.NewReader(body))
	rec := httptest.NewRecorder()

	s.handleMemoryAdd(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var addResp memoryAddResponse
	if err := json.NewDecoder(rec.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if addResp.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// List memories
	req = httptest.NewRequest("GET", "/api/memory", nil)
	rec = httptest.NewRecorder()

	s.handleMemoryList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var items []memoryItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Content != "likes Go" {
		t.Fatalf("expected 'likes Go', got %q", items[0].Content)
	}
}

func TestMemoryDelete(t *testing.T) {
	s := testServerWithMemoryDB(t)

	// Add a memory
	body := `{"content": "to be deleted", "category": "preference"}`
	addReq := httptest.NewRequest("POST", "/api/memory", strings.NewReader(body))
	addRec := httptest.NewRecorder()
	s.handleMemoryAdd(addRec, addReq)

	var addResp memoryAddResponse
	json.NewDecoder(addRec.Body).Decode(&addResp)

	// Delete it
	deleteReq := httptest.NewRequest("DELETE", "/api/memory/1", nil)
	deleteReq.SetPathValue("id", "1")
	deleteRec := httptest.NewRecorder()
	s.handleMemoryDelete(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	// Verify it's gone
	listReq := httptest.NewRequest("GET", "/api/memory", nil)
	listRec := httptest.NewRecorder()
	s.handleMemoryList(listRec, listReq)

	var items []memoryItem
	json.NewDecoder(listRec.Body).Decode(&items)
	if len(items) != 0 {
		t.Fatalf("expected 0 items after delete, got %d", len(items))
	}
}
