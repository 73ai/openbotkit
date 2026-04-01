package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// SkillInfo holds merged skill data for listing.
type SkillInfo struct {
	Name        string
	Source      string
	Description string
	Repo        string
}

// InstallCustomSkill installs a custom skill from provided content strings.
func InstallCustomSkill(name, skillContent, refContent string) error {
	return installCustom(name, skillContent, refContent, "custom", "")
}

// InstallExternalSkill installs a skill from an external repo.
func InstallExternalSkill(name, skillContent, refContent, repoURL string) error {
	return installCustom(name, skillContent, refContent, "external", repoURL)
}

func installCustom(name, skillContent, refContent, source, repo string) error {
	dir := filepath.Join(SkillsDir(), name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0600); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}

	if refContent != "" {
		if err := os.WriteFile(filepath.Join(dir, "REFERENCE.md"), []byte(refContent), 0600); err != nil {
			return fmt.Errorf("write REFERENCE.md: %w", err)
		}
	}

	manifest, err := LoadManifest()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	manifest.Skills[name] = SkillEntry{
		Source:  source,
		Version: "0.1.0",
		Repo:    repo,
	}
	if err := SaveManifest(manifest); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	return GenerateIndex()
}

// UpdateCustomSkill overwrites an existing custom or external skill.
func UpdateCustomSkill(name, skillContent, refContent string) error {
	manifest, err := LoadManifest()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	entry, ok := manifest.Skills[name]
	if !ok {
		return fmt.Errorf("skill %q not found", name)
	}
	if entry.Source != "custom" && entry.Source != "external" {
		return fmt.Errorf("cannot update %q skill %q (only custom/external skills can be updated)", entry.Source, name)
	}

	dir := filepath.Join(SkillsDir(), name)
	if skillContent != "" {
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillContent), 0600); err != nil {
			return fmt.Errorf("write SKILL.md: %w", err)
		}
	}
	if refContent != "" {
		if err := os.WriteFile(filepath.Join(dir, "REFERENCE.md"), []byte(refContent), 0600); err != nil {
			return fmt.Errorf("write REFERENCE.md: %w", err)
		}
	}

	return GenerateIndex()
}

// RemoveCustomSkill removes a custom or external skill.
func RemoveCustomSkill(name string) error {
	manifest, err := LoadManifest()
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	entry, ok := manifest.Skills[name]
	if !ok {
		return fmt.Errorf("skill %q not found", name)
	}
	if entry.Source != "custom" && entry.Source != "external" {
		return fmt.Errorf("cannot remove %q skill %q (only custom/external skills can be removed)", entry.Source, name)
	}

	dir := filepath.Join(SkillsDir(), name)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove skill dir: %w", err)
	}

	delete(manifest.Skills, name)
	if err := SaveManifest(manifest); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	return GenerateIndex()
}

// ListSkills returns all skills from the manifest merged with index data.
func ListSkills() ([]SkillInfo, error) {
	manifest, err := LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	index, err := LoadIndex()
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	descMap := make(map[string]string, len(index.Skills))
	for _, ie := range index.Skills {
		descMap[ie.Name] = ie.Description
	}

	var result []SkillInfo
	for name, entry := range manifest.Skills {
		result = append(result, SkillInfo{
			Name:        name,
			Source:      entry.Source,
			Description: descMap[name],
			Repo:        entry.Repo,
		})
	}
	return result, nil
}

// GetSkill reads the SKILL.md, REFERENCE.md, and manifest entry for a skill.
func GetSkill(name string) (skillMD, refMD string, entry SkillEntry, err error) {
	manifest, err := LoadManifest()
	if err != nil {
		return "", "", SkillEntry{}, fmt.Errorf("load manifest: %w", err)
	}

	entry, ok := manifest.Skills[name]
	if !ok {
		return "", "", SkillEntry{}, fmt.Errorf("skill %q not found", name)
	}

	dir := filepath.Join(SkillsDir(), name)
	skillData, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return "", "", entry, fmt.Errorf("read SKILL.md: %w", err)
	}
	skillMD = string(skillData)

	refData, err := os.ReadFile(filepath.Join(dir, "REFERENCE.md"))
	if err == nil {
		refMD = string(refData)
	}

	return skillMD, refMD, entry, nil
}
