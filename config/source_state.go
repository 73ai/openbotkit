package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SourceState is the per-source config stored at ~/.obk/<source>/config.json.
type SourceState struct {
	Linked bool `json:"linked"`
}

func sourceStatePath(name string) string {
	return filepath.Join(SourceDir(name), "config.json")
}

func LoadSourceState(name string) (*SourceState, error) {
	data, err := os.ReadFile(sourceStatePath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return &SourceState{}, nil
		}
		return nil, fmt.Errorf("read source state: %w", err)
	}
	var s SourceState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse source state: %w", err)
	}
	return &s, nil
}

func SaveSourceState(name string, state *SourceState) error {
	if err := EnsureSourceDir(name); err != nil {
		return fmt.Errorf("create source dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal source state: %w", err)
	}
	return os.WriteFile(sourceStatePath(name), data, 0600)
}

func IsSourceLinked(name string) bool {
	s, err := LoadSourceState(name)
	if err != nil {
		return false
	}
	return s.Linked
}

func LinkSource(name string) error {
	return SaveSourceState(name, &SourceState{Linked: true})
}

func UnlinkSource(name string) error {
	return SaveSourceState(name, &SourceState{Linked: false})
}
