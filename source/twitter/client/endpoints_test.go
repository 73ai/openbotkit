package client

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEndpoints_MissingFileFallsBackToDefaults(t *testing.T) {
	endpoints, err := LoadEndpoints(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(endpoints) == 0 {
		t.Fatal("expected default endpoints, got empty map")
	}
	ep, ok := endpoints["HomeTimeline"]
	if !ok {
		t.Fatal("expected HomeTimeline in defaults")
	}
	if ep.Method != "GET" {
		t.Errorf("HomeTimeline method = %q, want GET", ep.Method)
	}
	if ep.QueryID == "" {
		t.Error("HomeTimeline QueryID should not be empty")
	}
}

func TestLoadEndpoints_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "endpoints.yaml")
	content := `
HomeTimeline:
  query_id: "abc123"
  method: "GET"
CreateTweet:
  query_id: "xyz789"
  method: "POST"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	endpoints, err := LoadEndpoints(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints["HomeTimeline"].QueryID != "abc123" {
		t.Errorf("QueryID = %q, want abc123", endpoints["HomeTimeline"].QueryID)
	}
	if endpoints["CreateTweet"].Method != "POST" {
		t.Errorf("Method = %q, want POST", endpoints["CreateTweet"].Method)
	}
}

func TestLoadEndpoints_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("key: [invalid\n  broken:\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadEndpoints(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadEndpoints_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadEndpoints(path)
	if err == nil {
		t.Fatal("expected error for empty endpoints file")
	}
}

func TestDefaultEndpointsPath(t *testing.T) {
	path := DefaultEndpointsPath()
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if filepath.Base(path) != "endpoints.yaml" {
		t.Errorf("path base = %q, want endpoints.yaml", filepath.Base(path))
	}
}
