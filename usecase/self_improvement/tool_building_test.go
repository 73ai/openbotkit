package self_improvement

import (
	"context"
	"os"
	"os/exec"
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

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	workspaceDir := fx.WorkspaceDir()

	_, err := a.Run(ctx, "Build a simple bash script that counts the number of lines in a file. "+
		"Put the script at "+workspaceDir+"/tools/linecount/count.sh and make it executable. "+
		"Then create a skill called 'linecount' that teaches how to use it.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	// Verify script exists and is executable.
	scriptPath := filepath.Join(workspaceDir, "tools", "linecount", "count.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("script not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("script should be executable")
	}

	// Test the script works.
	testFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0600)
	out, err := exec.CommandContext(ctx, "bash", scriptPath, testFile).Output()
	if err != nil {
		t.Fatalf("run script: %v", err)
	}
	if !strings.Contains(string(out), "3") {
		t.Errorf("expected output to contain '3', got %q", string(out))
	}

	// Verify skill exists.
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if _, ok := m.Skills["linecount"]; !ok {
		t.Error("linecount skill not in manifest")
	}
}
