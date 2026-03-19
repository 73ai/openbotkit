package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultDirName = ".obk"
	ConfigFileName = "config.yaml"
)

// Dir returns the obk config directory.
// Checks OBK_CONFIG_DIR env var first, then falls back to ~/.obk/.
func Dir() string {
	if d := os.Getenv("OBK_CONFIG_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultDirName
	}
	return filepath.Join(home, DefaultDirName)
}

func FilePath() string {
	return filepath.Join(Dir(), ConfigFileName)
}

func SourceDir(sourceName string) string {
	return filepath.Join(Dir(), sourceName)
}

func EnsureDir() error {
	return os.MkdirAll(Dir(), 0700)
}

func EnsureSourceDir(sourceName string) error {
	return os.MkdirAll(SourceDir(sourceName), 0700)
}

func ModelsDir() string {
	return filepath.Join(Dir(), "models")
}

func ProviderDir(providerName string) string {
	return filepath.Join(Dir(), "providers", providerName)
}

func EnsureProviderDir(providerName string) error {
	return os.MkdirAll(ProviderDir(providerName), 0700)
}

func JobsDBPath() string {
	return filepath.Join(Dir(), "jobs.db")
}

func AuditDBPath() string {
	return filepath.Join(Dir(), "audit", "data.db")
}

func ScratchDir(sessionID string) string {
	return filepath.Join(Dir(), "scratch", sessionID)
}

func EnsureScratchDir(sessionID string) error {
	return os.MkdirAll(ScratchDir(sessionID), 0700)
}

func CleanScratch(sessionID string) error {
	return os.RemoveAll(ScratchDir(sessionID))
}

func LearningsDir() string {
	return filepath.Join(Dir(), "learnings")
}
