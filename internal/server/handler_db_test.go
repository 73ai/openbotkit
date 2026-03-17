package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	_ "modernc.org/sqlite"
)

func createTestDB(t *testing.T, dir, source string) {
	t.Helper()
	srcDir := filepath.Join(dir, source)
	if err := os.MkdirAll(srcDir, 0700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(srcDir, "data.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer db.Close()
	_, err = db.Exec("CREATE TABLE test (name TEXT, age INT); INSERT INTO test VALUES ('alice', 30); INSERT INTO test VALUES ('bob', 25);")
	if err != nil {
		t.Fatalf("seed test db: %v", err)
	}
}

func TestDBHandler_ValidQuery(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)
	createTestDB(t, dir, "gmail")

	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT name, age FROM test ORDER BY name"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dbResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Columns) != 2 || resp.Columns[0] != "name" {
		t.Fatalf("unexpected columns: %v", resp.Columns)
	}
	if len(resp.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(resp.Rows))
	}
	if resp.Rows[0][0] != "alice" {
		t.Fatalf("expected alice, got %q", resp.Rows[0][0])
	}
}

func TestDBHandler_UnknownSource(t *testing.T) {
	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT 1"}`
	req := httptest.NewRequest("POST", "/api/db/unknown", strings.NewReader(body))
	req.SetPathValue("source", "unknown")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDBHandler_NonSelectQuery(t *testing.T) {
	s := &Server{cfg: config.Default()}

	body := `{"sql": "DELETE FROM test"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDBHandler_StackedQueries(t *testing.T) {
	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT 1; DROP TABLE test"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDBHandler_EmptyResult(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	srcDir := filepath.Join(dir, "gmail")
	if err := os.MkdirAll(srcDir, 0700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(srcDir, "data.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Exec("CREATE TABLE test (name TEXT)")
	db.Close()

	s := &Server{cfg: config.Default()}

	body := `{"sql": "SELECT name FROM test"}`
	req := httptest.NewRequest("POST", "/api/db/gmail", strings.NewReader(body))
	req.SetPathValue("source", "gmail")
	rec := httptest.NewRecorder()

	s.handleDB(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dbResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Rows) != 0 {
		t.Fatalf("expected 0 rows, got %d", len(resp.Rows))
	}
}
