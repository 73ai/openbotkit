package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultIncludesWhatsApp(t *testing.T) {
	cfg := Default()
	if cfg.WhatsApp == nil {
		t.Fatal("expected WhatsApp config in defaults")
	}
	if cfg.WhatsApp.Storage.Driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %q", cfg.WhatsApp.Storage.Driver)
	}
}

func TestApplyDefaultsWhatsApp(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if cfg.WhatsApp == nil {
		t.Fatal("applyDefaults should create WhatsApp config")
	}
	if cfg.WhatsApp.Storage.Driver != "sqlite" {
		t.Fatalf("expected sqlite, got %q", cfg.WhatsApp.Storage.Driver)
	}
}

func TestWhatsAppDataDSNDefault(t *testing.T) {
	cfg := Default()
	dsn := cfg.WhatsAppDataDSN()
	if !strings.HasSuffix(dsn, filepath.Join("whatsapp", "data.db")) {
		t.Fatalf("expected path ending in whatsapp/data.db, got %q", dsn)
	}
}

func TestWhatsAppDataDSNCustom(t *testing.T) {
	cfg := Default()
	cfg.WhatsApp.Storage.DSN = "postgres://localhost/wa"
	dsn := cfg.WhatsAppDataDSN()
	if dsn != "postgres://localhost/wa" {
		t.Fatalf("expected custom DSN, got %q", dsn)
	}
}

func TestWhatsAppSessionDBPath(t *testing.T) {
	cfg := Default()
	path := cfg.WhatsAppSessionDBPath()
	if !strings.HasSuffix(path, filepath.Join("whatsapp", "session.db")) {
		t.Fatalf("expected path ending in whatsapp/session.db, got %q", path)
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `
whatsapp:
  storage:
    driver: postgres
    dsn: postgres://localhost/watest
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.WhatsApp.Storage.Driver != "postgres" {
		t.Fatalf("expected postgres, got %q", cfg.WhatsApp.Storage.Driver)
	}
	if cfg.WhatsApp.Storage.DSN != "postgres://localhost/watest" {
		t.Fatalf("expected custom dsn, got %q", cfg.WhatsApp.Storage.DSN)
	}
}

func TestLoadFromMissingFile(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("missing file should return defaults, got error: %v", err)
	}
	if cfg.WhatsApp == nil {
		t.Fatal("expected defaults to include WhatsApp")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	cfg.WhatsApp.Storage.Driver = "postgres"
	cfg.WhatsApp.Storage.DSN = "postgres://test"

	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.WhatsApp.Storage.Driver != "postgres" {
		t.Fatalf("expected postgres after reload, got %q", loaded.WhatsApp.Storage.Driver)
	}
	if loaded.WhatsApp.Storage.DSN != "postgres://test" {
		t.Fatalf("expected custom DSN after reload, got %q", loaded.WhatsApp.Storage.DSN)
	}
}
