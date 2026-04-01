package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
)

const testSkillMD = `---
name: test-skill
description: A test skill for unit testing
allowed-tools: Bash(obk *)
---

This is a test skill.
`

const testRefMD = `## Commands

` + "```bash" + `
obk test run
` + "```" + `
`

func TestInstallCustomSkill(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if err := InstallCustomSkill("test-skill", testSkillMD, testRefMD); err != nil {
		t.Fatalf("InstallCustomSkill: %v", err)
	}

	// Verify files exist.
	dir := filepath.Join(SkillsDir(), "test-skill")
	skillData, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if string(skillData) != testSkillMD {
		t.Errorf("SKILL.md content mismatch")
	}

	refData, err := os.ReadFile(filepath.Join(dir, "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if string(refData) != testRefMD {
		t.Errorf("REFERENCE.md content mismatch")
	}

	// Verify manifest entry.
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	entry, ok := m.Skills["test-skill"]
	if !ok {
		t.Fatal("test-skill not in manifest")
	}
	if entry.Source != "custom" {
		t.Errorf("expected source 'custom', got %q", entry.Source)
	}

	// Verify index contains the skill.
	idx, err := LoadIndex()
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	found := false
	for _, ie := range idx.Skills {
		if ie.Name == "test-skill" {
			found = true
			if ie.Description != "A test skill for unit testing" {
				t.Errorf("unexpected description: %q", ie.Description)
			}
		}
	}
	if !found {
		t.Error("test-skill not found in index")
	}
}

func TestInstallCustomSkill_WithCaution(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	ref := `## Warning

> [!CAUTION]
> This skill performs destructive operations.

## Commands
` + "```bash" + `
obk dangerous-thing
` + "```" + `
`
	if err := InstallCustomSkill("danger-skill", testSkillMD, ref); err != nil {
		t.Fatalf("InstallCustomSkill: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(SkillsDir(), "danger-skill", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if !strings.Contains(string(data), "[!CAUTION]") {
		t.Error("expected [!CAUTION] marker in REFERENCE.md")
	}
}

func TestInstallExternalSkill(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if err := InstallExternalSkill("ext-skill", testSkillMD, testRefMD, "https://github.com/example/repo"); err != nil {
		t.Fatalf("InstallExternalSkill: %v", err)
	}

	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	entry := m.Skills["ext-skill"]
	if entry.Source != "external" {
		t.Errorf("expected source 'external', got %q", entry.Source)
	}
	if entry.Repo != "https://github.com/example/repo" {
		t.Errorf("expected repo URL, got %q", entry.Repo)
	}
}

func TestUpdateCustomSkill(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if err := InstallCustomSkill("test-skill", testSkillMD, testRefMD); err != nil {
		t.Fatalf("install: %v", err)
	}

	newRef := "## Updated\n\nNew reference content.\n"
	if err := UpdateCustomSkill("test-skill", "", newRef); err != nil {
		t.Fatalf("UpdateCustomSkill: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(SkillsDir(), "test-skill", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if string(data) != newRef {
		t.Errorf("REFERENCE.md not updated, got %q", string(data))
	}
}

func TestUpdateCustomSkill_RejectsBuiltin(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	// Install builtin skills to populate manifest.
	if _, err := Install(config.Default()); err != nil {
		t.Fatalf("Install: %v", err)
	}

	err := UpdateCustomSkill("email-read", "new content", "new ref")
	if err == nil {
		t.Fatal("expected error when updating builtin skill")
	}
	if !strings.Contains(err.Error(), "obk") {
		t.Errorf("error should mention source type, got: %v", err)
	}
}

func TestRemoveCustomSkill(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if err := InstallCustomSkill("test-skill", testSkillMD, testRefMD); err != nil {
		t.Fatalf("install: %v", err)
	}

	if err := RemoveCustomSkill("test-skill"); err != nil {
		t.Fatalf("RemoveCustomSkill: %v", err)
	}

	// Dir removed.
	if _, err := os.Stat(filepath.Join(SkillsDir(), "test-skill")); !os.IsNotExist(err) {
		t.Error("skill directory should be removed")
	}

	// Manifest updated.
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if _, ok := m.Skills["test-skill"]; ok {
		t.Error("test-skill should not be in manifest after removal")
	}

	// Index updated.
	idx, err := LoadIndex()
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	for _, ie := range idx.Skills {
		if ie.Name == "test-skill" {
			t.Error("test-skill should not be in index after removal")
		}
	}
}

func TestRemoveCustomSkill_RejectsBuiltin(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if _, err := Install(config.Default()); err != nil {
		t.Fatalf("Install: %v", err)
	}

	err := RemoveCustomSkill("email-read")
	if err == nil {
		t.Fatal("expected error when removing builtin skill")
	}
}

func TestListSkills(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if err := InstallCustomSkill("skill-a", testSkillMD, testRefMD); err != nil {
		t.Fatalf("install skill-a: %v", err)
	}
	skill2 := strings.Replace(testSkillMD, "test-skill", "skill-b", 1)
	skill2 = strings.Replace(skill2, "A test skill for unit testing", "Second skill", 1)
	if err := InstallCustomSkill("skill-b", skill2, testRefMD); err != nil {
		t.Fatalf("install skill-b: %v", err)
	}

	list, err := ListSkills()
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(list))
	}

	names := make(map[string]bool)
	for _, s := range list {
		names[s.Name] = true
		if s.Source != "custom" {
			t.Errorf("expected source 'custom' for %s, got %q", s.Name, s.Source)
		}
	}
	if !names["skill-a"] || !names["skill-b"] {
		t.Errorf("expected both skill-a and skill-b in list")
	}
}

func TestGetSkill(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	if err := InstallCustomSkill("test-skill", testSkillMD, testRefMD); err != nil {
		t.Fatalf("install: %v", err)
	}

	skillMD, refMD, entry, err := GetSkill("test-skill")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if skillMD != testSkillMD {
		t.Errorf("SKILL.md content mismatch")
	}
	if refMD != testRefMD {
		t.Errorf("REFERENCE.md content mismatch")
	}
	if entry.Source != "custom" {
		t.Errorf("expected source 'custom', got %q", entry.Source)
	}
}

func TestInstallPreservesCustomSkills(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	// Install custom skill first.
	if err := InstallCustomSkill("my-custom", testSkillMD, testRefMD); err != nil {
		t.Fatalf("install custom: %v", err)
	}

	// Run declarative Install() — should not wipe custom skill.
	if _, err := Install(config.Default()); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Verify custom skill survives.
	m, err := LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	entry, ok := m.Skills["my-custom"]
	if !ok {
		t.Fatal("custom skill was removed by Install()")
	}
	if entry.Source != "custom" {
		t.Errorf("custom skill source changed to %q", entry.Source)
	}

	// Verify files still on disk.
	data, err := os.ReadFile(filepath.Join(SkillsDir(), "my-custom", "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if len(data) == 0 {
		t.Error("SKILL.md is empty after Install()")
	}
}

func TestInstallCustomSkill_RejectsPathTraversal(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cases := []string{
		"../escape",
		"../../etc/evil",
		"foo/bar",
		".hidden",
		"-starts-with-dash",
		"has spaces",
		"UPPERCASE",
		"",
	}
	for _, name := range cases {
		if err := InstallCustomSkill(name, testSkillMD, testRefMD); err == nil {
			t.Errorf("expected error for skill name %q, got nil", name)
		}
	}
}

func TestInstallCustomSkill_AcceptsValidNames(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	cases := []string{"my-skill", "sqlite3-query", "a", "tool-v2"}
	for _, name := range cases {
		if err := InstallCustomSkill(name, testSkillMD, testRefMD); err != nil {
			t.Errorf("unexpected error for skill name %q: %v", name, err)
		}
	}
}
