package config

import (
	"path/filepath"
	"testing"
)

func TestSetByPath_TopLevelString(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "workspace", "/custom/workspace"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Workspace != "/custom/workspace" {
		t.Fatalf("expected /custom/workspace, got %q", cfg.Workspace)
	}
}

func TestSetByPath_TopLevelTimezone(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "timezone", "America/New_York"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Timezone != "America/New_York" {
		t.Fatalf("expected America/New_York, got %q", cfg.Timezone)
	}
}

func TestSetByPath_NestedPath(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "gmail.storage.driver", "postgres"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Gmail.Storage.Driver != "postgres" {
		t.Fatalf("expected postgres, got %q", cfg.Gmail.Storage.Driver)
	}
}

func TestSetByPath_InitializesNilPointer(t *testing.T) {
	cfg := &Config{}
	if err := SetByPath(cfg, "daemon.gmail_sync_period", "30m"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Daemon == nil {
		t.Fatal("Daemon should have been allocated")
	}
	if cfg.Daemon.GmailSyncPeriod != "30m" {
		t.Fatalf("expected 30m, got %q", cfg.Daemon.GmailSyncPeriod)
	}
}

func TestSetByPath_Bool(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "gmail.download_attachments", "true"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if !cfg.Gmail.DownloadAttachments {
		t.Fatal("expected DownloadAttachments to be true")
	}

	if err := SetByPath(cfg, "gmail.download_attachments", "false"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Gmail.DownloadAttachments {
		t.Fatal("expected DownloadAttachments to be false")
	}
}

func TestSetByPath_Int(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "gmail.sync_days", "30"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Gmail.SyncDays != 30 {
		t.Fatalf("expected 30, got %d", cfg.Gmail.SyncDays)
	}
}

func TestSetByPath_InvalidPath(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "nonexistent.field", "value"); err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestSetByPath_InvalidNestedPath(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "gmail.storage.nonexistent", "value"); err == nil {
		t.Fatal("expected error for invalid nested path")
	}
}

func TestSetByPath_EmptyPath(t *testing.T) {
	cfg := Default()
	if err := SetByPath(cfg, "", "value"); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestSetByPath_DeepNested(t *testing.T) {
	cfg := &Config{}
	if err := SetByPath(cfg, "providers.google.credentials_file", "/path/to/creds.json"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if cfg.Providers == nil || cfg.Providers.Google == nil {
		t.Fatal("nested pointers should have been allocated")
	}
	if cfg.Providers.Google.CredentialsFile != "/path/to/creds.json" {
		t.Fatalf("expected /path/to/creds.json, got %q", cfg.Providers.Google.CredentialsFile)
	}
}

func TestConfigSetWorkspace_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	if err := SetByPath(cfg, "workspace", "/tmp/my-workspace"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Workspace != "/tmp/my-workspace" {
		t.Fatalf("expected /tmp/my-workspace, got %q", loaded.Workspace)
	}
}

func TestConfigSetTimezone_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	if err := SetByPath(cfg, "timezone", "America/Los_Angeles"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Timezone != "America/Los_Angeles" {
		t.Fatalf("expected America/Los_Angeles, got %q", loaded.Timezone)
	}
}

func TestConfigSetNested_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	if err := SetByPath(cfg, "gmail.storage.driver", "postgres"); err != nil {
		t.Fatalf("SetByPath: %v", err)
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Gmail.Storage.Driver != "postgres" {
		t.Fatalf("expected postgres, got %q", loaded.Gmail.Storage.Driver)
	}
}
