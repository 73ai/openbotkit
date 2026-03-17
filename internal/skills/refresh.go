package skills

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/73ai/openbotkit/config"
)

// RefreshGWSSkills checks if gws skills need updating and re-resolves if so.
// Triggers include: gws version change, scope change.
func RefreshGWSSkills(cfg *config.Config) error {
	if cfg.Integrations == nil || cfg.Integrations.GWS == nil || !cfg.Integrations.GWS.Enabled {
		return nil
	}

	gwsPath, err := exec.LookPath("gws")
	if err != nil {
		return nil // gws not installed, nothing to refresh
	}

	manifest, err := LoadManifest()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	currentVersion := gwsVersion(gwsPath)
	manifestVersion := gwsManifestVersion(manifest)

	if currentVersion == manifestVersion {
		slog.Debug("gws skills up to date", "version", currentVersion)
		return nil
	}

	slog.Info("gws skills outdated, refreshing", "manifest", manifestVersion, "current", currentVersion)
	_, err = Install(cfg)
	if err != nil {
		return fmt.Errorf("refresh gws skills: %w", err)
	}
	return nil
}

func gwsManifestVersion(m *Manifest) string {
	for _, entry := range m.Skills {
		if entry.Source == "gws" {
			return strings.TrimSpace(entry.Version)
		}
	}
	return ""
}
