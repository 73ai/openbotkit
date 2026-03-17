package skills

import (
	"testing"

	"github.com/73ai/openbotkit/config"
)

func TestRefreshGWSSkills_GWSDisabled(t *testing.T) {
	cfg := &config.Config{}
	if err := RefreshGWSSkills(cfg); err != nil {
		t.Fatalf("RefreshGWSSkills: %v", err)
	}
}

func TestRefreshGWSSkills_GWSNotOnPath(t *testing.T) {
	cfg := &config.Config{
		Integrations: &config.IntegrationsConfig{
			GWS: &config.GWSConfig{Enabled: true, Services: []string{"calendar"}},
		},
	}
	t.Setenv("PATH", "/nonexistent")
	if err := RefreshGWSSkills(cfg); err != nil {
		t.Fatalf("RefreshGWSSkills: %v", err)
	}
}

func TestGWSManifestVersion(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillEntry{
			"gws-calendar": {Source: "gws", Version: "1.2.3"},
			"email-read":   {Source: "obk", Version: "0.1.0"},
		},
	}
	if v := gwsManifestVersion(m); v != "1.2.3" {
		t.Errorf("version = %q, want 1.2.3", v)
	}
}

func TestGWSManifestVersion_NoGWS(t *testing.T) {
	m := &Manifest{
		Skills: map[string]SkillEntry{
			"email-read": {Source: "obk", Version: "0.1.0"},
		},
	}
	if v := gwsManifestVersion(m); v != "" {
		t.Errorf("version = %q, want empty", v)
	}
}
