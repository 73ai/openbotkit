package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestCheckConfig_MissingFile(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	results := checkConfig(nil, nil)
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].Status != "FAIL" {
		t.Fatalf("expected FAIL for missing config, got %s", results[0].Status)
	}
}

func TestCheckConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("mode: local\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	results := checkConfig(cfg, nil)
	if results[0].Status != "OK" {
		t.Fatalf("expected OK for valid config, got %s", results[0].Status)
	}
}

func TestCheckDatabases_NoneExist(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	cfg := config.Default()
	results := checkDatabases(cfg)
	for _, r := range results {
		if r.Status != "WARN" {
			t.Fatalf("expected WARN for %s, got %s", r.Name, r.Status)
		}
	}
}

func TestCheckDatabases_AllExpectedNamesPresent(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	cfg := config.Default()
	results := checkDatabases(cfg)

	expected := map[string]bool{
		"Gmail DB":      false,
		"WhatsApp DB":   false,
		"History DB":    false,
		"UserMemory DB": false,
		"AppleNotes DB": false,
		"Contacts DB":   false,
		"WebSearch DB":  false,
		"iMessage DB":   false,
		"Scheduler DB":  false,
		"Audit DB":      false,
		"Jobs DB":       false,
	}
	for _, r := range results {
		if _, ok := expected[r.Name]; ok {
			expected[r.Name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing database check: %s", name)
		}
	}
}

func TestCheckDatabases_SomeExist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	gmailDir := filepath.Join(dir, "gmail")
	os.MkdirAll(gmailDir, 0700)
	os.WriteFile(filepath.Join(gmailDir, "data.db"), []byte("x"), 0600)

	cfg := config.Default()
	results := checkDatabases(cfg)

	found := false
	for _, r := range results {
		if r.Name == "Gmail DB" {
			found = true
			if r.Status != "OK" {
				t.Fatalf("expected OK for Gmail DB, got %s", r.Status)
			}
		}
	}
	if !found {
		t.Fatal("Gmail DB not found in results")
	}
}

func TestCheckGoogleOAuth_Missing(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	cfg := config.Default()
	results := checkGoogleOAuth(cfg)
	if len(results) != 1 || results[0].Status != "WARN" {
		t.Fatalf("expected WARN, got %v", results)
	}
}

func TestCheckWhatsAppSession_Missing(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	cfg := config.Default()
	results := checkWhatsAppSession(cfg)
	if len(results) != 1 || results[0].Status != "WARN" {
		t.Fatalf("expected WARN, got %v", results)
	}
}

func TestCheckSkills_NoIndex(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	result := checkSkills()
	if result.Status != "WARN" {
		t.Fatalf("expected WARN for missing skills, got %s", result.Status)
	}
}

func TestCheckAPIKeys_NilModels(t *testing.T) {
	cfg := config.Default()
	results := checkAPIKeys(cfg)
	if results != nil {
		t.Fatalf("expected nil for nil Models, got %v", results)
	}
}
