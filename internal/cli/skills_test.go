package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/internal/skills"
	"github.com/spf13/cobra"
)

func writeTestFiles(t *testing.T, dir string) (skillFile, refFile string) {
	t.Helper()
	skillFile = filepath.Join(dir, "SKILL.md")
	refFile = filepath.Join(dir, "REFERENCE.md")
	skillContent := "---\nname: test-cli-skill\ndescription: CLI test skill\nallowed-tools: Bash(obk *)\n---\n\nA test skill.\n"
	refContent := "## Commands\n\n```bash\nobk test run\n```\n"
	if err := os.WriteFile(skillFile, []byte(skillContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(refFile, []byte(refContent), 0600); err != nil {
		t.Fatal(err)
	}
	return
}

// newCreateCmd returns a fresh create command with its own flag set.
func newCreateCmd() *cobra.Command {
	cmd := *skillsCreateCmd
	cmd.ResetFlags()
	cmd.Flags().String("skill-file", "", "Path to SKILL.md file")
	cmd.Flags().String("ref-file", "", "Path to REFERENCE.md file")
	return &cmd
}

// newUpdateCmd returns a fresh update command with its own flag set.
func newUpdateCmd() *cobra.Command {
	cmd := *skillsUpdateCmd
	cmd.ResetFlags()
	cmd.Flags().String("skill-file", "", "Path to updated SKILL.md file")
	cmd.Flags().String("ref-file", "", "Path to updated REFERENCE.md file")
	return &cmd
}

func TestSkillsCreate(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	staging := t.TempDir()
	skillFile, refFile := writeTestFiles(t, staging)

	cmd := newCreateCmd()
	cmd.Flags().Set("skill-file", skillFile)
	cmd.Flags().Set("ref-file", refFile)
	if err := cmd.RunE(cmd, []string{"test-cli-skill"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Verify via list.
	list, err := skills.ListSkills()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, s := range list {
		if s.Name == "test-cli-skill" {
			found = true
		}
	}
	if !found {
		t.Error("test-cli-skill not found in list")
	}

	// Verify via show.
	skillMD, _, _, err := skills.GetSkill("test-cli-skill")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(skillMD, "CLI test skill") {
		t.Error("SKILL.md does not contain expected description")
	}
}

func TestSkillsCreate_NoRefFile(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	staging := t.TempDir()
	skillFile, _ := writeTestFiles(t, staging)

	cmd := newCreateCmd()
	cmd.Flags().Set("skill-file", skillFile)
	if err := cmd.RunE(cmd, []string{"no-ref-skill"}); err != nil {
		t.Fatalf("create without ref: %v", err)
	}

	_, refMD, _, err := skills.GetSkill("no-ref-skill")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if refMD != "" {
		t.Errorf("expected empty REFERENCE.md, got %q", refMD)
	}
}

func TestSkillsUpdate(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	staging := t.TempDir()
	skillFile, refFile := writeTestFiles(t, staging)

	cmd := newCreateCmd()
	cmd.Flags().Set("skill-file", skillFile)
	cmd.Flags().Set("ref-file", refFile)
	if err := cmd.RunE(cmd, []string{"update-skill"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Write updated files.
	newRef := filepath.Join(staging, "NEW_REF.md")
	os.WriteFile(newRef, []byte("## Updated\n\nNew content.\n"), 0600)

	ucmd := newUpdateCmd()
	ucmd.Flags().Set("ref-file", newRef)
	if err := ucmd.RunE(ucmd, []string{"update-skill"}); err != nil {
		t.Fatalf("update: %v", err)
	}

	_, refMD, _, err := skills.GetSkill("update-skill")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(refMD, "New content") {
		t.Errorf("REFERENCE.md not updated: %q", refMD)
	}
}

func TestSkillsUpdate_NonexistentFails(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	staging := t.TempDir()
	_, refFile := writeTestFiles(t, staging)

	ucmd := newUpdateCmd()
	ucmd.Flags().Set("ref-file", refFile)
	err := ucmd.RunE(ucmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error updating nonexistent skill")
	}
}

func TestSkillsRemove(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	staging := t.TempDir()
	skillFile, refFile := writeTestFiles(t, staging)

	cmd := newCreateCmd()
	cmd.Flags().Set("skill-file", skillFile)
	cmd.Flags().Set("ref-file", refFile)
	cmd.RunE(cmd, []string{"remove-me"})

	if err := skillsRemoveCmd.RunE(skillsRemoveCmd, []string{"remove-me"}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	list, _ := skills.ListSkills()
	for _, s := range list {
		if s.Name == "remove-me" {
			t.Error("skill should be removed from list")
		}
	}
}

func TestSkillsRemove_BuiltinFails(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	// Install builtins.
	skills.Install(config.Default())

	err := skillsRemoveCmd.RunE(skillsRemoveCmd, []string{"email-read"})
	if err == nil {
		t.Fatal("expected error removing builtin skill")
	}
}

func TestSkillsShow_NotFound(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	err := skillsShowCmd.RunE(skillsShowCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestSkillsListEmpty(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := skillsListCmd.RunE(skillsListCmd, nil)

	w.Close()
	os.Stdout = origStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Should have header but no data rows (or just the header).
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) > 1 {
		t.Errorf("expected only header, got %d lines", len(lines))
	}
}

func TestSkillsRoundTrip(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	staging := t.TempDir()
	skillFile, refFile := writeTestFiles(t, staging)

	// Create.
	cmd := newCreateCmd()
	cmd.Flags().Set("skill-file", skillFile)
	cmd.Flags().Set("ref-file", refFile)
	if err := cmd.RunE(cmd, []string{"round-trip"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// List — should contain it.
	list, _ := skills.ListSkills()
	found := false
	for _, s := range list {
		if s.Name == "round-trip" {
			found = true
		}
	}
	if !found {
		t.Fatal("skill not in list after create")
	}

	// Show — should have content.
	skillMD, refMD, _, err := skills.GetSkill("round-trip")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if skillMD == "" {
		t.Error("SKILL.md empty")
	}
	if refMD == "" {
		t.Error("REFERENCE.md empty")
	}

	// Update.
	newRef := filepath.Join(staging, "UPDATED.md")
	os.WriteFile(newRef, []byte("## V2\n\nUpdated.\n"), 0600)
	ucmd := newUpdateCmd()
	ucmd.Flags().Set("ref-file", newRef)
	if err := ucmd.RunE(ucmd, []string{"round-trip"}); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Show — should have updated content.
	_, refMD, _, _ = skills.GetSkill("round-trip")
	if !strings.Contains(refMD, "V2") {
		t.Error("REFERENCE.md not updated")
	}

	// Remove.
	if err := skillsRemoveCmd.RunE(skillsRemoveCmd, []string{"round-trip"}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// List — should not contain it.
	list, _ = skills.ListSkills()
	for _, s := range list {
		if s.Name == "round-trip" {
			t.Error("skill still in list after remove")
		}
	}
}
