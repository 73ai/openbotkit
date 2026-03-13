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

func TestRequireSetup_NoModels(t *testing.T) {
	cfg := Default()
	if err := cfg.RequireSetup(); err == nil {
		t.Fatal("expected error when models not configured")
	}
}

func TestRequireSetup_NoDefault(t *testing.T) {
	cfg := Default()
	cfg.Models = &ModelsConfig{}
	if err := cfg.RequireSetup(); err == nil {
		t.Fatal("expected error when default model not set")
	}
}

func TestRequireSetup_OK(t *testing.T) {
	cfg := Default()
	cfg.Models = &ModelsConfig{Default: "anthropic/claude-sonnet-4-6"}
	if err := cfg.RequireSetup(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvedMode_DefaultsToLocal(t *testing.T) {
	cfg := Default()
	if cfg.ResolvedMode() != ModeLocal {
		t.Fatalf("expected local, got %q", cfg.ResolvedMode())
	}
}

func TestResolvedMode_ExplicitValues(t *testing.T) {
	tests := []struct {
		mode Mode
		want Mode
	}{
		{ModeLocal, ModeLocal},
		{ModeRemote, ModeRemote},
		{ModeServer, ModeServer},
	}
	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			cfg := Default()
			cfg.Mode = tt.mode
			if cfg.ResolvedMode() != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, cfg.ResolvedMode())
			}
		})
	}
}

func TestIsLocal_IsRemote_IsServer(t *testing.T) {
	cfg := Default()
	if !cfg.IsLocal() {
		t.Fatal("default should be local")
	}
	if cfg.IsRemote() {
		t.Fatal("default should not be remote")
	}
	if cfg.IsServer() {
		t.Fatal("default should not be server")
	}

	cfg.Mode = ModeRemote
	if cfg.IsLocal() {
		t.Fatal("remote should not be local")
	}
	if !cfg.IsRemote() {
		t.Fatal("remote should be remote")
	}

	cfg.Mode = ModeServer
	if !cfg.IsServer() {
		t.Fatal("server should be server")
	}
}

func TestSchedulerDataDSNDefault(t *testing.T) {
	cfg := Default()
	dsn := cfg.SchedulerDataDSN()
	if !strings.HasSuffix(dsn, filepath.Join("scheduler", "data.db")) {
		t.Fatalf("expected path ending in scheduler/data.db, got %q", dsn)
	}
}

func TestSchedulerDataDSNCustom(t *testing.T) {
	cfg := Default()
	cfg.Scheduler.Storage.DSN = "postgres://localhost/sched"
	dsn := cfg.SchedulerDataDSN()
	if dsn != "postgres://localhost/sched" {
		t.Fatalf("expected custom DSN, got %q", dsn)
	}
}

func TestApplyDefaultsScheduler(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()
	if cfg.Scheduler == nil {
		t.Fatal("applyDefaults should create Scheduler config")
	}
	if cfg.Scheduler.Storage.Driver != "sqlite" {
		t.Fatalf("expected sqlite, got %q", cfg.Scheduler.Storage.Driver)
	}
}

func TestSourceDataDSN_ValidSources(t *testing.T) {
	cfg := Default()
	sources := []struct {
		name   string
		suffix string
	}{
		{"gmail", filepath.Join("gmail", "data.db")},
		{"whatsapp", filepath.Join("whatsapp", "data.db")},
		{"history", filepath.Join("history", "data.db")},
		{"user_memory", filepath.Join("user_memory", "data.db")},
		{"applenotes", filepath.Join("applenotes", "data.db")},
		{"imessage", filepath.Join("imessage", "data.db")},
		{"websearch", filepath.Join("websearch", "data.db")},
		{"scheduler", filepath.Join("scheduler", "data.db")},
	}
	for _, s := range sources {
		t.Run(s.name, func(t *testing.T) {
			dsn, err := cfg.SourceDataDSN(s.name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasSuffix(dsn, s.suffix) {
				t.Fatalf("expected path ending in %s, got %q", s.suffix, dsn)
			}
		})
	}
}

func TestSourceDataDSN_UnknownSource(t *testing.T) {
	cfg := Default()
	_, err := cfg.SourceDataDSN("unknown")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestConfigRoundTrip_WithModeAndRemote(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	cfg.Mode = ModeRemote
	cfg.Remote = &RemoteConfig{
		Server:   "https://example.com:8443",
		Username: "testuser",
		Password: "testpass",
	}
	cfg.Auth = &AuthConfig{
		Username: "admin",
		Password: "secret",
	}
	cfg.Channels = &ChannelsConfig{
		Telegram: &TelegramConfig{
			BotToken: "123:abc",
			OwnerID:  42,
		},
	}

	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Mode != ModeRemote {
		t.Fatalf("expected remote mode, got %q", loaded.Mode)
	}
	if loaded.Remote == nil || loaded.Remote.Server != "https://example.com:8443" {
		t.Fatal("remote config not preserved")
	}
	if loaded.Auth == nil || loaded.Auth.Username != "admin" {
		t.Fatal("auth config not preserved")
	}
	if loaded.Channels == nil || loaded.Channels.Telegram == nil || loaded.Channels.Telegram.OwnerID != 42 {
		t.Fatal("channels config not preserved")
	}
}

func TestBackwardCompat_NoModeField(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `
gmail:
  storage:
    driver: sqlite
`
	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.IsLocal() {
		t.Fatalf("old config without mode should default to local, got %q", cfg.ResolvedMode())
	}
}

func TestWebSearchDataDSN(t *testing.T) {
	cfg := Default()
	dsn := cfg.WebSearchDataDSN()
	if !strings.HasSuffix(dsn, filepath.Join("websearch", "data.db")) {
		t.Fatalf("expected path ending in websearch/data.db, got %q", dsn)
	}
}

func TestWebSearchDataDSNCustom(t *testing.T) {
	cfg := Default()
	cfg.WebSearch.Storage.DSN = "postgres://localhost/ws"
	dsn := cfg.WebSearchDataDSN()
	if dsn != "postgres://localhost/ws" {
		t.Fatalf("expected custom DSN, got %q", dsn)
	}
}

func TestApplyDefaultsWebSearch(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if cfg.WebSearch == nil {
		t.Fatal("applyDefaults should create WebSearch config")
	}
	if cfg.WebSearch.Storage.Driver != "sqlite" {
		t.Fatalf("expected sqlite, got %q", cfg.WebSearch.Storage.Driver)
	}
	if cfg.WebSearch.Timeout != "15s" {
		t.Fatalf("expected 15s timeout, got %q", cfg.WebSearch.Timeout)
	}
}

func TestGWSCallbackURL_Nil(t *testing.T) {
	cfg := Default()
	if url := cfg.GWSCallbackURL(); url != "" {
		t.Fatalf("expected empty, got %q", url)
	}
}

func TestGWSCallbackURL_IntegrationsSetGWSNil(t *testing.T) {
	cfg := Default()
	cfg.Integrations = &IntegrationsConfig{}
	if url := cfg.GWSCallbackURL(); url != "" {
		t.Fatalf("expected empty when GWS is nil, got %q", url)
	}
}

func TestGWSCallbackURL_Set(t *testing.T) {
	cfg := Default()
	cfg.Integrations = &IntegrationsConfig{
		GWS: &GWSConfig{
			CallbackURL: "https://example.ngrok-free.app/auth/google/callback",
		},
	}
	want := "https://example.ngrok-free.app/auth/google/callback"
	if got := cfg.GWSCallbackURL(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGWSConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	cfg.Integrations = &IntegrationsConfig{
		GWS: &GWSConfig{
			Enabled:     true,
			Services:    []string{"drive", "docs"},
			CallbackURL: "https://example.ngrok-free.app/auth/google/callback",
			NgrokDomain: "example.ngrok-free.app",
		},
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Integrations == nil || loaded.Integrations.GWS == nil {
		t.Fatal("GWS config not preserved")
	}
	if loaded.Integrations.GWS.CallbackURL != cfg.Integrations.GWS.CallbackURL {
		t.Fatalf("callback URL not preserved: %q", loaded.Integrations.GWS.CallbackURL)
	}
	if loaded.Integrations.GWS.NgrokDomain != cfg.Integrations.GWS.NgrokDomain {
		t.Fatalf("ngrok domain not preserved: %q", loaded.Integrations.GWS.NgrokDomain)
	}
}

func TestModelsConfig_CompactionFields_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	cfg.Models = &ModelsConfig{
		Default:             "gemini/gemini-2.5-flash",
		ContextWindow:       150000,
		CompactionThreshold: 0.25,
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Models.ContextWindow != 150000 {
		t.Fatalf("ContextWindow = %d, want 150000", loaded.Models.ContextWindow)
	}
	if loaded.Models.CompactionThreshold != 0.25 {
		t.Fatalf("CompactionThreshold = %f, want 0.25", loaded.Models.CompactionThreshold)
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
