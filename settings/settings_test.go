package settings

import (
	"fmt"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
)

func testService(cfg *config.Config) *Service {
	creds := make(map[string]string)
	return New(cfg,
		WithSaveFn(func(*config.Config) error { return nil }),
		WithStoreCred(func(ref, value string) error {
			creds[ref] = value
			return nil
		}),
		WithLoadCred(func(ref string) (string, error) {
			v, ok := creds[ref]
			if !ok {
				return "", nil
			}
			return v, nil
		}),
	)
}

func TestGetSetMode(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "mode")
	if field == nil {
		t.Fatal("mode field not found")
	}

	got := svc.GetValue(field)
	if got != "local" {
		t.Errorf("default mode = %q, want %q", got, "local")
	}

	if err := svc.SetValue(field, "remote"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "remote" {
		t.Errorf("after set, mode = %q, want %q", got, "remote")
	}
}

func TestGetSetTimezone(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "timezone")
	if field == nil {
		t.Fatal("timezone field not found")
	}

	if err := svc.SetValue(field, "Asia/Ho_Chi_Minh"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "Asia/Ho_Chi_Minh" {
		t.Errorf("timezone = %q, want %q", got, "Asia/Ho_Chi_Minh")
	}
}

func TestTimezoneValidation(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "timezone")
	if field == nil {
		t.Fatal("timezone field not found")
	}

	if err := svc.SetValue(field, "Not/A/Timezone"); err == nil {
		t.Error("expected validation error for invalid timezone")
	}

	if err := svc.SetValue(field, ""); err != nil {
		t.Errorf("empty timezone should be valid: %v", err)
	}
}

func TestGetSetNilModels(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "models.default")
	if field == nil {
		t.Fatal("models.default field not found")
	}

	got := svc.GetValue(field)
	if got != "" {
		t.Errorf("nil models default = %q, want empty", got)
	}

	if err := svc.SetValue(field, "anthropic/claude-sonnet-4-6"); err != nil {
		t.Fatal(err)
	}
	if cfg.Models == nil {
		t.Fatal("Models should be initialized after set")
	}
	if got := svc.GetValue(field); got != "anthropic/claude-sonnet-4-6" {
		t.Errorf("default model = %q, want %q", got, "anthropic/claude-sonnet-4-6")
	}
}

func TestGetSetBool(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "gmail.download_attachments")
	if field == nil {
		t.Fatal("gmail.download_attachments field not found")
	}

	if err := svc.SetValue(field, "true"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "true" {
		t.Errorf("download_attachments = %q, want %q", got, "true")
	}
}

func TestGetSetNumber(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "gmail.sync_days")
	if field == nil {
		t.Fatal("gmail.sync_days field not found")
	}

	if err := svc.SetValue(field, "30"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "30" {
		t.Errorf("sync_days = %q, want %q", got, "30")
	}

	if err := svc.SetValue(field, "abc"); err == nil {
		t.Error("expected validation error for non-numeric value")
	}
}

func TestPasswordField(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.providers.anthropic.api_key")
	if field == nil {
		t.Fatal("anthropic api_key field not found")
	}

	got := svc.GetValue(field)
	if got != "not configured" {
		t.Errorf("initial api key status = %q, want %q", got, "not configured")
	}

	if err := svc.SetValue(field, "sk-test-key"); err != nil {
		t.Fatal(err)
	}
	got = svc.GetValue(field)
	if got != "configured" {
		t.Errorf("after set, api key status = %q, want %q", got, "configured")
	}
}

func TestNilChannels(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "channels.telegram.bot_token")
	if field == nil {
		t.Fatal("telegram bot_token field not found")
	}

	got := svc.GetValue(field)
	if got != "not configured" {
		t.Errorf("nil channels bot_token = %q, want %q", got, "not configured")
	}

	if err := svc.SetValue(field, "123:ABC"); err != nil {
		t.Fatal(err)
	}
	if cfg.Channels == nil || cfg.Channels.Telegram == nil {
		t.Fatal("Channels.Telegram should be initialized after set")
	}
}

func TestTreeStructure(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	tree := svc.Tree()
	if len(tree) != 6 {
		t.Fatalf("tree has %d top-level nodes, want 6", len(tree))
	}

	labels := []string{"General", "Models", "Channels", "Data Sources", "Integrations", "Advanced"}
	for i, n := range tree {
		if n.Category == nil {
			t.Errorf("tree[%d] is not a category", i)
			continue
		}
		if n.Category.Label != labels[i] {
			t.Errorf("tree[%d].Label = %q, want %q", i, n.Category.Label, labels[i])
		}
	}
}

func TestSaveError(t *testing.T) {
	cfg := config.Default()
	svc := New(cfg,
		WithSaveFn(func(*config.Config) error {
			return fmt.Errorf("disk full")
		}),
	)

	field := findField(svc, "timezone")
	if field == nil {
		t.Fatal("timezone field not found")
	}

	err := svc.SetValue(field, "UTC")
	if err == nil {
		t.Fatal("expected save error")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "disk full")
	}
}

func TestCredentialNilHandlers(t *testing.T) {
	cfg := config.Default()
	svc := New(cfg,
		WithSaveFn(func(*config.Config) error { return nil }),
	)

	if err := svc.StoreCredential("ref", "val"); err == nil {
		t.Error("expected error from nil storeCred")
	}
	if _, err := svc.LoadCredential("ref"); err == nil {
		t.Error("expected error from nil loadCred")
	}
}

func TestSelectProfileRoundTrip(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.profile")
	if field == nil {
		t.Fatal("models.profile field not found")
	}

	if err := svc.SetValue(field, "gemini"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "gemini" {
		t.Errorf("profile = %q, want %q", got, "gemini")
	}
}

func TestAuthMethodSelectRoundTrip(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.providers.anthropic.auth_method")
	if field == nil {
		t.Fatal("anthropic auth_method field not found")
	}

	got := svc.GetValue(field)
	if got != "api_key" {
		t.Errorf("default auth_method = %q, want %q", got, "api_key")
	}

	if err := svc.SetValue(field, "vertex_ai"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "vertex_ai" {
		t.Errorf("auth_method = %q, want %q", got, "vertex_ai")
	}
}

func TestContextWindowRoundTrip(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.context_window")
	if field == nil {
		t.Fatal("models.context_window field not found")
	}

	if got := svc.GetValue(field); got != "" {
		t.Errorf("default context_window = %q, want empty", got)
	}

	if err := svc.SetValue(field, "128000"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "128000" {
		t.Errorf("context_window = %q, want %q", got, "128000")
	}

	if err := svc.SetValue(field, ""); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "" {
		t.Errorf("cleared context_window = %q, want empty", got)
	}
}

func TestCompactionThresholdRoundTrip(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.compaction_threshold")
	if field == nil {
		t.Fatal("models.compaction_threshold field not found")
	}

	if err := svc.SetValue(field, "0.8"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "0.8" {
		t.Errorf("compaction_threshold = %q, want %q", got, "0.8")
	}
}

func TestNilIntegrationsGWS(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "integrations.gws.enabled")
	if field == nil {
		t.Fatal("gws.enabled field not found")
	}

	if got := svc.GetValue(field); got != "false" {
		t.Errorf("nil gws enabled = %q, want %q", got, "false")
	}

	if err := svc.SetValue(field, "true"); err != nil {
		t.Fatal(err)
	}
	if cfg.Integrations == nil || cfg.Integrations.GWS == nil {
		t.Fatal("Integrations.GWS should be initialized after set")
	}
	if got := svc.GetValue(field); got != "true" {
		t.Errorf("gws.enabled = %q, want %q", got, "true")
	}
}

func TestNilWebSearch(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "websearch.proxy")
	if field == nil {
		t.Fatal("websearch.proxy field not found")
	}

	if got := svc.GetValue(field); got != "" {
		t.Errorf("nil websearch proxy = %q, want empty", got)
	}

	if err := svc.SetValue(field, "socks5://localhost:1080"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "socks5://localhost:1080" {
		t.Errorf("proxy = %q, want %q", got, "socks5://localhost:1080")
	}
}

func TestNilDaemon(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "daemon.gmail_sync_period")
	if field == nil {
		t.Fatal("daemon.gmail_sync_period field not found")
	}

	if got := svc.GetValue(field); got != "" {
		t.Errorf("nil daemon sync period = %q, want empty", got)
	}

	if err := svc.SetValue(field, "30m"); err != nil {
		t.Fatal(err)
	}
	if cfg.Daemon == nil {
		t.Fatal("Daemon should be initialized after set")
	}
	if got := svc.GetValue(field); got != "30m" {
		t.Errorf("sync period = %q, want %q", got, "30m")
	}
}

func TestTelegramOwnerID(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	field := findField(svc, "channels.telegram.owner_id")
	if field == nil {
		t.Fatal("telegram owner_id field not found")
	}

	if got := svc.GetValue(field); got != "" {
		t.Errorf("nil owner_id = %q, want empty", got)
	}

	if err := svc.SetValue(field, "123456789"); err != nil {
		t.Fatal(err)
	}
	if got := svc.GetValue(field); got != "123456789" {
		t.Errorf("owner_id = %q, want %q", got, "123456789")
	}

	if err := svc.SetValue(field, "not-a-number"); err == nil {
		t.Error("expected validation error for non-numeric owner_id")
	}
}

func TestPasswordEmptySetIsNoop(t *testing.T) {
	cfg := config.Default()
	svc := testService(cfg)

	field := findField(svc, "models.providers.openai.api_key")
	if field == nil {
		t.Fatal("openai api_key field not found")
	}

	if err := svc.SetValue(field, ""); err != nil {
		t.Fatal(err)
	}

	if got := svc.GetValue(field); got != "not configured" {
		t.Errorf("after empty set, status = %q, want %q", got, "not configured")
	}
}

func TestPasswordLoadCredFailure(t *testing.T) {
	cfg := &config.Config{
		Models: &config.ModelsConfig{
			Providers: map[string]config.ModelProviderConfig{
				"openai": {APIKeyRef: "keychain:obk/openai"},
			},
		},
	}
	svc := New(cfg,
		WithSaveFn(func(*config.Config) error { return nil }),
		WithLoadCred(func(ref string) (string, error) {
			return "", fmt.Errorf("keychain locked")
		}),
	)

	field := findField(svc, "models.providers.openai.api_key")
	if field == nil {
		t.Fatal("openai api_key field not found")
	}

	got := svc.GetValue(field)
	if !strings.Contains(got, "configured") {
		t.Errorf("loadCred failure status = %q, want to contain %q", got, "configured")
	}
	if !strings.Contains(got, "ref:") {
		t.Errorf("loadCred failure status = %q, want to contain ref fallback", got)
	}
}

func TestGWSCallbackAndNgrok(t *testing.T) {
	cfg := &config.Config{}
	svc := testService(cfg)

	cbField := findField(svc, "integrations.gws.callback_url")
	ngField := findField(svc, "integrations.gws.ngrok_domain")
	if cbField == nil || ngField == nil {
		t.Fatal("gws callback_url or ngrok_domain field not found")
	}

	if err := svc.SetValue(cbField, "https://example.com/callback"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetValue(ngField, "my-app.ngrok.io"); err != nil {
		t.Fatal(err)
	}

	if got := svc.GetValue(cbField); got != "https://example.com/callback" {
		t.Errorf("callback_url = %q, want %q", got, "https://example.com/callback")
	}
	if got := svc.GetValue(ngField); got != "my-app.ngrok.io" {
		t.Errorf("ngrok_domain = %q, want %q", got, "my-app.ngrok.io")
	}
}

func findField(svc *Service, key string) *Field {
	return findFieldInNodes(svc.Tree(), key)
}

func findFieldInNodes(nodes []Node, key string) *Field {
	for _, n := range nodes {
		if n.Field != nil && n.Field.Key == key {
			return n.Field
		}
		if n.Category != nil {
			if f := findFieldInNodes(n.Category.Children, key); f != nil {
				return f
			}
		}
	}
	return nil
}
