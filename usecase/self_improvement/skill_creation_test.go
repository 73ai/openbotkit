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

	_, err := a.Run(ctx, "Create a skill called 'sqlite-query' that helps me query local SQLite databases. The skill should teach how to use the sqlite3 CLI to query any .db file. Include examples for listing tables, querying data, and showing schema.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	// Verify SKILL.md exists with correct frontmatter.
	skillDir := filepath.Join(fx.Dir(), "skills", "sqlite-query")
	skillData, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	if !strings.Contains(string(skillData), "name: sqlite-query") {
		t.Error("SKILL.md should contain name: sqlite-query in frontmatter")
	}

	// Verify REFERENCE.md exists with sqlite3 examples.
	refData, err := os.ReadFile(filepath.Join(skillDir, "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if !strings.Contains(string(refData), "sqlite3") {
		t.Error("REFERENCE.md should contain sqlite3 examples")
	}

	// Verify manifest has source: custom.
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	entry, ok := m.Skills["sqlite-query"]
	if !ok {
		t.Fatal("sqlite-query not in manifest")
	}
	if entry.Source != "custom" {
		t.Errorf("expected source 'custom', got %q", entry.Source)
	}

	// Verify index contains the skill.
	idx, err := skills.LoadIndex()
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	found := false
	for _, ie := range idx.Skills {
		if ie.Name == "sqlite-query" {
			found = true
		}
	}
	if !found {
		t.Error("sqlite-query not found in skill index")
	}
}

func TestUseCase_UpdateCustomSkill(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// First create the skill.
	_, err := a.Run(ctx, "Create a skill called 'db-query' that helps query SQLite databases using sqlite3 CLI.")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Then update it.
	_, err = a.Run(ctx, "Update the 'db-query' skill to also support PostgreSQL via the psql CLI. Add psql connection and query examples.")
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify REFERENCE.md now mentions psql.
	refData, err := os.ReadFile(filepath.Join(fx.Dir(), "skills", "db-query", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if !strings.Contains(string(refData), "psql") {
		t.Error("REFERENCE.md should contain psql after update")
	}
}

func TestUseCase_CreateSkillWithCaution(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Create a skill called 'file-cleanup' for deleting temporary files from the filesystem. Mark it as dangerous since it performs destructive operations — it should have a CAUTION marker.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	refData, err := os.ReadFile(filepath.Join(fx.Dir(), "skills", "file-cleanup", "REFERENCE.md"))
	if err != nil {
		t.Fatalf("read REFERENCE.md: %v", err)
	}
	if !strings.Contains(string(refData), "[!CAUTION]") {
		t.Error("REFERENCE.md should contain [!CAUTION] marker")
	}
}
