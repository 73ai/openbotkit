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

func TestUseCase_BuildBashTool(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	workspaceDir := fx.WorkspaceDir()

	// Two-step: first build the tool, then create the skill.
	_, err := a.Run(ctx, "Write a bash script that counts lines in a file (using wc -l) and save it to "+workspaceDir+"/tools/linecount.sh. Make it executable.")
	if err != nil {
		t.Fatalf("build tool: %v", err)
	}

	_, err = a.Run(ctx, "Create a skill that teaches you how to use the linecount tool you just built.")
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}

	// Verify a script exists somewhere in the workspace.
	scriptPath := findScript(t, workspaceDir)
	if scriptPath == "" {
		// Log what's in workspace for debugging.
		filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				rel, _ := filepath.Rel(workspaceDir, path)
				t.Logf("workspace: %s (mode=%s, size=%d)", rel, info.Mode(), info.Size())
			}
			return nil
		})
		t.Fatal("no script found in workspace")
	}
	t.Logf("script at: %s", scriptPath)

	// Verify a custom skill was created.
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	foundCustom := false
	for _, entry := range m.Skills {
		if entry.Source == "custom" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Error("no custom skill was created for the tool")
	}
}

// findScript walks the directory looking for a script file.
func findScript(t *testing.T, dir string) string {
	t.Helper()
	var found string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		isScript := ext == ".sh" || ext == ".bash" || ext == ".py"
		isExec := info.Mode()&0111 != 0
		if isScript || isExec {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
