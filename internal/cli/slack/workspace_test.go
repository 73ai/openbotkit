package slack

import (
	"testing"

	"github.com/73ai/openbotkit/config"
	slacksrc "github.com/73ai/openbotkit/source/slack"
	"github.com/73ai/openbotkit/source/slack/desktop"
	"github.com/zalando/go-keyring"
)

func multiWorkspaceConfig() *config.SlackConfig {
	return &config.SlackConfig{
		DefaultWorkspace: "acme",
		Workspaces: map[string]config.SlackWorkspace{
			"acme":      {TeamName: "Acme"},
			"beta-corp": {TeamName: "Beta Corp"},
		},
	}
}

func TestResolveWorkspace(t *testing.T) {
	cfg := multiWorkspaceConfig()
	tests := []struct {
		name      string
		explicit  string
		subdomain string
		want      string
	}{
		{"default", "", "", "acme"},
		{"explicit", "beta-corp", "", "beta-corp"},
		{"explicit sanitized", "Beta Corp", "", "beta-corp"},
		{"explicit beats permalink", "acme", "betacorp", "acme"},
		{"permalink subdomain", "", "betacorp", "beta-corp"},
		{"unknown subdomain falls back to default", "", "other", "acme"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveWorkspace(cfg, tt.explicit, tt.subdomain)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveWorkspace_Errors(t *testing.T) {
	if _, err := resolveWorkspace(nil, "", ""); err == nil {
		t.Error("expected error for nil config")
	}
	if _, err := resolveWorkspace(multiWorkspaceConfig(), "nope", ""); err == nil {
		t.Error("expected error for unknown workspace")
	}
	noDefault := multiWorkspaceConfig()
	noDefault.DefaultWorkspace = ""
	if _, err := resolveWorkspace(noDefault, "", ""); err == nil {
		t.Error("expected error without default")
	}
}

func TestAddDesktopWorkspaces(t *testing.T) {
	keyring.MockInit()
	cfg := &config.Config{}
	creds := []*desktop.Credentials{
		{Token: "xoxc-a", Cookie: "xoxd-c", TeamID: "T1", TeamName: "Acme"},
		{Token: "xoxc-b", Cookie: "xoxd-c", TeamID: "T2", TeamName: "Beta Corp"},
	}

	names, err := AddDesktopWorkspaces(cfg, creds)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "acme" || names[1] != "beta-corp" {
		t.Fatalf("names = %v", names)
	}
	if cfg.Slack.DefaultWorkspace != "acme" {
		t.Errorf("default = %q", cfg.Slack.DefaultWorkspace)
	}
	if ws := cfg.Slack.Workspaces["beta-corp"]; ws.TeamID != "T2" || ws.AuthMode != "desktop" {
		t.Errorf("beta-corp = %+v", ws)
	}
	got, err := slacksrc.LoadCredentials("beta-corp")
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != "xoxc-b" || got.Cookie != "xoxd-c" {
		t.Errorf("creds = %+v", got)
	}
}

func TestAddDesktopWorkspaces_KeepsExistingDefault(t *testing.T) {
	keyring.MockInit()
	cfg := &config.Config{Slack: &config.SlackConfig{
		DefaultWorkspace: "old",
		Workspaces:       map[string]config.SlackWorkspace{"old": {}},
	}}
	if _, err := AddDesktopWorkspaces(cfg, []*desktop.Credentials{{Token: "xoxc-a", TeamName: "Acme"}}); err != nil {
		t.Fatal(err)
	}
	if cfg.Slack.DefaultWorkspace != "old" || len(cfg.Slack.Workspaces) != 2 {
		t.Errorf("slack cfg = %+v", cfg.Slack)
	}
}
