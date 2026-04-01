package self_improvement

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/internal/skills"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_InstallFromLocalRepo(t *testing.T) {
	fx := usecase.NewFixture(t)
	skillExtra := fx.LoadSkillContent(t, "skill-creator")
	a := fx.Agent(t, skillExtra)

	// Create a local git repo with a Python script and README.
	repoDir := t.TempDir()
	setupLocalRepo(t, repoDir)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Install skills from this local repo: "+repoDir+
		". It has a Python script that converts CSV to JSON. "+
		"Read the README to understand it, then create a skill for it.")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	// Verify a skill was created (name may vary — agent decides the name).
	m, err := skills.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	foundCustom := false
	for _, entry := range m.Skills {
		if entry.Source == "custom" || entry.Source == "external" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Error("expected at least one custom/external skill after install")
	}

	// Verify staging dir is cleaned up (agent should have cleaned it).
	stagingDir := filepath.Join(fx.WorkspaceDir(), "staging")
	entries, _ := os.ReadDir(stagingDir)
	for _, e := range entries {
		if e.IsDir() && e.Name() != "." {
			// Repo clone should be removed.
			t.Logf("note: staging dir still has %q (agent may not have cleaned up)", e.Name())
		}
	}
}

func setupLocalRepo(t *testing.T, dir string) {
	t.Helper()

	// Create README.
	readme := `# CSV to JSON Converter

A simple Python script that converts CSV files to JSON format.

## Usage

` + "```bash" + `
python3 convert.py input.csv output.json
` + "```" + `

## Arguments
- input.csv: Path to the input CSV file
- output.json: Path to the output JSON file
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0600); err != nil {
		t.Fatal(err)
	}

	// Create Python script.
	script := `#!/usr/bin/env python3
import csv
import json
import sys

def convert(input_path, output_path):
    with open(input_path, 'r') as f:
        reader = csv.DictReader(f)
        rows = list(reader)
    with open(output_path, 'w') as f:
        json.dump(rows, f, indent=2)
    print(f"Converted {len(rows)} rows")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: convert.py <input.csv> <output.json>")
        sys.exit(1)
    convert(sys.argv[1], sys.argv[2])
`
	if err := os.WriteFile(filepath.Join(dir, "convert.py"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize as a git repo.
	cmds := [][]string{
		{"git", "init"},
		{"git", "add", "."},
		{"git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
