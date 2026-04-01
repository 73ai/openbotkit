package self_improvement

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/internal/skills"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_CreateCustomSkill(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "I often need to query local SQLite databases. Create a skill that teaches you how to use sqlite3 to explore and query .db files — listing tables, running SELECT queries, showing schema, that kind of thing.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	// Find the custom skill the agent created (we don't dictate the name).
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var skillName string
	for name, entry := range m.Skills {
		if entry.Source == "custom" {
			skillName = name
			break
		}
	}
	if skillName == "" {
		t.Fatal("no custom skill was created in manifest")
	}

	// Verify SKILL.md exists with frontmatter.
	skillDir := filepath.Join(fx.Dir(), "skills", skillName)
	skillData, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md for %s: %v", skillName, err)
	}
	if !strings.Contains(string(skillData), "---") {
		t.Error("SKILL.md should contain YAML frontmatter")
	}

	// Verify REFERENCE.md exists with sqlite3 content.
	refData, err := os.ReadFile(filepath.Join(skillDir, "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md for %s: %v", skillName, err)
	}
	if !strings.Contains(strings.ToLower(string(refData)), "sqlite") {
		t.Error("REFERENCE.md should mention sqlite")
	}

	// Verify the skill appears in the index (searchable).
	idx, err := skills.LoadIndex()
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	found := false
	for _, ie := range idx.Skills {
		if ie.Name == skillName {
			found = true
		}
	}
	if !found {
		t.Errorf("skill %q not found in index", skillName)
	}
}

func TestUseCase_UpdateCustomSkill(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// First create a skill.
	_, err := a.Run(ctx, "Create a skill for querying SQLite databases using sqlite3.")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Find the skill that was created.
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var skillName string
	for name, entry := range m.Skills {
		if entry.Source == "custom" {
			skillName = name
			break
		}
	}
	if skillName == "" {
		t.Fatal("no custom skill was created")
	}

	// Ask the agent to update it — use the name it chose.
	_, err = a.Run(ctx, "Update the '"+skillName+"' skill to also cover PostgreSQL via psql. Add connection and query examples for psql.")
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify REFERENCE.md now mentions psql.
	refData, err := os.ReadFile(filepath.Join(fx.Dir(), "skills", skillName, "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(refData)), "psql") {
		t.Error("REFERENCE.md should contain psql after update")
	}
}

func TestUseCase_CreateSkillWithCaution(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Create a skill for cleaning up temporary files from directories. This is destructive — it deletes files — so make sure it has appropriate safety warnings.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	// Find the custom skill.
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var skillName string
	for name, entry := range m.Skills {
		if entry.Source == "custom" {
			skillName = name
			break
		}
	}
	if skillName == "" {
		t.Fatal("no custom skill was created")
	}

	refData, err := os.ReadFile(filepath.Join(fx.Dir(), "skills", skillName, "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if !strings.Contains(string(refData), "[!CAUTION]") {
		t.Error("REFERENCE.md should contain [!CAUTION] marker for destructive skill")
	}
}
